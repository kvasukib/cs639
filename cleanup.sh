#!/bin/bash

ssh mumble-01 "killall master"&
ssh mumble-40 "killall serv"
ssh mumble-39 "killall serv"
ssh mumble-38 "killall serv"