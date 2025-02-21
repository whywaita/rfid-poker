#!/bin/bash -x

# [{A spades} {Q clubs}]
go run main.go -serial "00:00:5e:00:53:00" -cards "040e3bd2286b85,040f43d2286b85" -host "http://localhost:8080"
#
# [{K clubs} {6 diamonds}]
go run main.go -serial "00:00:5e:00:53:01" -cards "04101b9a776b85,041248d2286b85" -host "http://localhost:8080"
#
## [{K spades} {8 hearts} {6 hearts}]
#go run main.go -serial "00:00:5e:00:53:00" -cards "04143ad2286b85,04173bd2286b85,041a3bd2286b85" -host "http://localhost:8081"

# [{A hearts}]
#go run main.go -serial "00:00:5e:00:53:00" -cards "041a48d2286b85" -host "http://localhost:8080"
#
# [{T diamonds}]
#go run main.go -serial "00:00:5e:00:53:00" -cards "041e48d2286b85" -host "http://localhost:8080"
#
## [{J hearts}]
#go run main.go -serial "00:00:5e:00:53:01" -cards "041f43d2286b85" -host "http://localhost:8080"
#
##[{2 diamonds}]
#go run main.go -serial "00:00:5e:00:53:01" -cards "04263bd2286b85" -host "http://localhost:8080"

# muck [{A spades} {Q clubs}]
#go run main.go -serial "00:00:5e:00:53:02" -cards "040e3bd2286b85,040f43d2286b85" -host "http://localhost:8080"
