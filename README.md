This no longer works with Android 11+ because now a random port is chosen to connect. Don't use it anymore.

----

# adc
`adc` is a small command that does a quick port scan to find your Android phone in your network.


### Use-Case 
If you often connect your phone to your PC using "ADB over network", then you might know that it is annoying to type in the IP/name of your phone. This tool automatically searches your network for phones with an open adb debug port and connects to the first one found.

This basically replaces adb. Instead of writing `adb connect ...` and `adb shell`, you just type `adc shell`.
If your phone is already connected to the adb daemon, adc no longer scans the network and redirects the command quite fast.
Any parameters not recognized by this program will be given to adb after a phone has connected.

### Installation
Install [Go](https://golang.org/dl/). You can now go-get this program using

    go get -u github.com/xarantolus/adc

This should install the command and move it into your $PATH.

### [License](LICENSE)
This is free as in freedom software. Do whatever you like with it.
