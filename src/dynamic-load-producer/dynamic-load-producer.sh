#!/bin/bash

# Function to determine requests per minute based on the current hour

get_requests_per_minute() {
    local hour=$(date +%H)
    local max_requests_per_minute=65000000 #rough maximum number of request that a single frontend-demo can handle without overloading
    local percentage
    case $hour in
        00|01|02|03|04|05) percentage=10 ;;  # 00:00 - 05:59
        06|07) percentage=40 ;;              # 06:00 - 07:59
        08|09) percentage=50 ;;              # 08:00 - 09:59
        10|11) percentage=90 ;;              # 10:00 - 11:59
        12|13|14) percentage=110 ;;          # 12:00 - 14:59
        15|16) percentage=90 ;;              # 15:00 - 16:59
        17|18) percentage=75 ;;              # 17:00 - 18:59
        19|20) percentage=40 ;;              # 19:00 - 20:59
        21|22|23) percentage=20 ;;           # 21:00 - 23:59
    esac
    echo $(( (percentage * max_requests_per_minute) / 100 ))
}

# Main loop
while true; do
    start_second=$(date +%s)
    requests_per_minute=$(get_requests_per_minute)

    echo "load_producer_target_requests_per_minute $requests_per_minute" | \
          curl --data-binary @- http://prometheus:9091/metrics/job/dynamic-load-producer

    curl -s -o /dev/null -X GET --connect-timeout 2 "http://frontend-demo:8080/heavyLoad?iters=$requests_per_minute"

    end_second=$(date +%s)
    elapsed=$(( end_second - start_second ))
    sleep_time=$(( 60 - elapsed ))
    if [ $sleep_time -gt 0 ]; then
        sleep $sleep_time
    fi
done
