package cmd

import "os"

const terraformReleasesUrl = "https://releases.hashicorp.com/terraform"

var rootUrl = os.Getenv("HOME") + "/.tfenvgo"
var terraformBinPath = rootUrl + "/bin"
var terraformVersionPath = rootUrl + "/versions"
var arch = "amd64"
var osType = "linux"



// Colors
var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Magenta = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"
