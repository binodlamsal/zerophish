#!/bin/bash
PATH=/sbin:/bin:/usr/sbin:/usr/bin:/usr/local/sbin:/usr/local/bin
processName=Phishing-Simulation
process=phishing-simulation
appDirectory=/root/go/src/github.com/everycloud-technologies/phishing-simulation
logfile=/var/log/phishing-simulation/phishing-simulation.log
errfile=/var/log/phishing-simulation/phishing-simulation.error

readpassphrase() {
	if [[ -v PRODUCTION ]]; then
		read -sp 'Enter passphrase: ' pass && echo $pass > input.pipe
	fi
}

start() {
	echo 'Starting '${processName}'…'
	cd ${appDirectory}
	readpassphrase
	sleep 1
	nohup ./$process >>$logfile 2>>$errfile &
	sleep 1
	status
}

encryptemails () {
	cd ${appDirectory}
	./$process --encrypt-emails
	sleep 1
}

decryptemails () {
	cd ${appDirectory}
	./$process --decrypt-emails
	sleep 1
}

decryptapikeys () {
	cd ${appDirectory}
	./$process --decrypt-api-keys
	sleep 1
}


encryptapikeys () {
	cd ${appDirectory}
	./$process --encrypt-api-keys
	sleep 1
}


stop() {
	echo 'Stopping '${processName}'…'
	kill `pidof ${process}`
	sleep 1
}

restart(){
	stop
	start
}

status() {
	pid=$(pidof ${process})
	if [[ "$pid" != "" ]]; then
	echo ${processName}' is running…'
	else
	echo ${processName}' is not running…'
	tail $errfile
	fi
}

case $1 in
start|stop|status|encryptemails|encryptapikeys|decryptemails|decryptapikeys) "$1" ;;
esac
