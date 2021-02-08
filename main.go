package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	flagPort = flag.Int("p", 5555, "Port to scan for")

	flagTimeout = flag.Duration("t", 30*time.Second, "Scan timeout")
)

func main() {
	flag.Parse()

	cmd := exec.Command("adb", "start-server")

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	cmd.Run()

	ctx, cancel := context.WithTimeout(context.Background(), *flagTimeout)
	defer cancel()

	var connected bool
start_count:
	count, err := deviceCount(ctx)
	if err != nil {
		fmt.Println("error getting device count:", err.Error())
		os.Exit(1)
	}
	if count == 0 {
		connect(ctx)
		connected = true
		goto start_count
	} else if count > 1 {
		fmt.Println("More than one device connected")
		os.Exit(1)
	}

	// Now execute the actual command we were given, e.g. when calling "adc shell"
	if args := flag.Args(); len(args) > 0 {
		cmd := exec.Command("adb", args...)

		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin

		err := cmd.Run()
		if err != nil {
			ee, ok := err.(*exec.ExitError)
			if ok {
				os.Exit(ee.ExitCode())
			}
			os.Exit(1)
		}
	} else if !connected {
		fmt.Println("Connected")
	}
}

func deviceCount(ctx context.Context) (count int, err error) {
	cmd := exec.CommandContext(ctx, "adb", "devices", "-l")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")

	for i, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if i == 0 && l == "List of devices attached" {
			continue
		}
		if i > 0 && strings.HasPrefix(l, "*") {
			continue
		}

		count++
	}

	return
}

func connect(ctx context.Context) {
	prefixes, err := findPrefixes()
	if err != nil {
		panic(err)
	}

	var port = strconv.Itoa(*flagPort)

	scanCtx, cancel := context.WithCancel(ctx)

	rand.Seed(time.Now().Unix())

	var list []int
	for i := 0; i <= 255; i++ {
		list = append(list, i)
	}

	rand.Shuffle(len(list), func(i, j int) {
		list[i], list[j] = list[j], list[i]
	})

	var result = make(chan string)

	fmt.Println("Starting scan...")
	for i := range list {
		for _, pref := range prefixes {
			go scan(scanCtx, pref+strconv.Itoa(i)+":"+port, result)
		}
	}

forloop:
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Error: timeout")
			os.Exit(1)
		case ip := <-result:
			// If one is done, we cancel all others
			cancel()

			fmt.Println("Connected to", ip)
			break forloop
		}
	}
}

func scan(ctx context.Context, ip string, result chan string) {
	conn, err := net.DialTimeout("tcp", ip, *flagTimeout/3)
	if err != nil {
		return
	}
	conn.Close()

	connectTimeout, c := context.WithCancel(ctx)
	defer c()

	err = exec.CommandContext(connectTimeout, "adb", "connect", ip).Run()
	if err != nil {
		return
	}

	// Wait until no longer unauthorized
	time.Sleep(1 * time.Second)

	err = exec.CommandContext(connectTimeout, "adb", "shell", "echo").Run()
	if err != nil {
		exec.Command("adb", "disconnect", ip).Run()
		return
	}

	result <- ip
}

func findPrefixes() (prefixes []string, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			split := strings.Split(ip.String(), ".")
			if len(split) != 4 {
				continue
			}

			prefixes = append(prefixes, strings.Join(split[:3], ".")+".")
		}
	}
	if len(prefixes) == 0 {
		err = fmt.Errorf("Couldn't find any network prefixes")
	}
	return
}
