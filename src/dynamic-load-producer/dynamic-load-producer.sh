#!/bin/bash

# Function to determine requests per minute based on the current hour

get_requests_per_minute() {
    local hour=$(date +%H)
    local scaling_factor=10000
    local base_rpm
    case $hour in
        00|01|02|03|04|05) base_rpm=22 ;;    # 00:00 - 05:59
        06|07) base_rpm=520 ;;               # 06:00 - 07:59
        08|09) base_rpm=780 ;;               # 08:00 - 09:59
        10|11) base_rpm=1000 ;;              # 10:00 - 11:59
        12|13|14) base_rpm=1600 ;;           # 12:00 - 14:59
        15|16) base_rpm=1150 ;;              # 15:00 - 16:59
        17|18) base_rpm=800 ;;               # 17:00 - 18:59
        19|20) base_rpm=400 ;;               # 19:00 - 20:59
        21|22|23) base_rpm=350 ;;            # 21:00 - 23:59
    esac
    echo "($base_rpm * $scaling_factor) / 1" | bc
}

# Main loop
while true; do
    start_second=$(date +%s)
    requests_per_minute=$(get_requests_per_minute)

    echo "load_producer_target_requests_per_minute $requests_per_minute" | \
          curl --data-binary @- http://prometheus:9091/metrics/job/dynamic-load-producer

    curl -s -o /dev/null -X GET "http://frontend-demo:8080/heavyLoad?iters=$requests_per_minute"

    end_second=$(date +%s)
    elapsed=$(( end_second - start_second ))
    sleep_time=$(( 60 - elapsed ))
    if [ $sleep_time -gt 0 ]; then
        sleep $sleep_time
    fi
done
