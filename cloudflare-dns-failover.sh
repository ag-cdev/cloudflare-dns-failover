#!/bin/bash

set -e

CF_API_TOKEN="myToken" # Cloudflare API token
CF_ZONE_ID="myZoneID" # Cloudflare Zone ID
CF_RECORD="example.com" # DNS record to update
SERVER_IPS="ip1,ip2,ip3" # List of server IPs in order of priority
CHECK_INTERVAL=10 # Interval between checks in seconds

for tool in curl jq; do
	if ! command -v "$tool" &> /dev/null; then
		echo "ERROR: $tool not installed"
		exit 1
	fi
done

# Logs message without consecutive repetition since script is running in a loop
log-no-repeat() {
	if [[ "$lastlog" != "$1" ]]; then
		echo "$(date +"[%Y-%m-%d %H:%M:%S]"): $1"
		lastlog="$1"
	fi
}

update_record() {
	local ip=$1
	result=$(curl -s \
		-X PUT "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/dns_records/$record_id" \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $CF_API_TOKEN" \
		--data "{\"type\":\"A\",\"name\":\"$CF_RECORD\",\"content\":\"$ip\",\"ttl\":1,\"proxied\":false}" |
		jq -r .success)

	if [[ $result == "true" ]]; then
		log-no-repeat "$CF_RECORD updated to: $ip (previous: $cf_ip)"
		cf_ip="$ip" # Update A record IP variable to reflect the change in our main loop
	else
		log-no-repeat "ERROR: $CF_RECORD update failed!"
	fi
}

initial_data=$(curl -s \
	-X GET "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/dns_records?type=A&name=$CF_RECORD" \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $CF_API_TOKEN")

record_id=$(jq -r '.result[0].id' <<< "$initial_data") # Extract record ID for use in updates later
cf_ip=$(jq -r '.result[0].content' <<< "$initial_data") # Extract current A record's IP

# Main loop to check IP responsiveness and update A record if necessary
while true; do
	for ip in ${SERVER_IPS//,/ }; do
		if ping -q -W5 -c1 "$ip" &> /dev/null; then
			if [[ "$cf_ip" != "$ip" ]]; then
				update_record "$ip"
				break
			else
				log-no-repeat "$CF_RECORD already set to $ip"
				break
			fi
		fi
	done

	# After trying all IPs, log error if no responsive IP was set as the current one
	if [[ "$cf_ip" != "$ip" ]]; then
		log-no-repeat "ERROR: No responsive IP found!"
	fi

	sleep $CHECK_INTERVAL
done
