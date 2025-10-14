#!/bin/sh

# wait-for-deps ensures dependent services are reachable before starting the app.

set -e

RETRIES="${WAIT_FOR_RETRIES:-60}"
SLEEP_INTERVAL="${WAIT_FOR_INTERVAL:-2}"

wait_for_service() {
	service_name="$1"
	host="$2"
	port="$3"
	count=0

	if [ -z "$host" ] || [ -z "$port" ]; then
		echo "Skipping $service_name check; host or port not provided"
		return 0
	fi

	echo "Waiting for $service_name at ${host}:${port}"
	while ! nc -z "$host" "$port" >/dev/null 2>&1; do
		count=$((count + 1))
		if [ "$count" -ge "$RETRIES" ]; then
			echo "Timed out waiting for $service_name (host: ${host}, port: ${port}) after $((RETRIES * SLEEP_INTERVAL)) seconds"
			exit 1
		fi
		sleep "$SLEEP_INTERVAL"
	done
	echo "$service_name is available"
}

# PostgreSQL
wait_for_service "PostgreSQL" "${POSTGRES_HOST:-postgres}" "${POSTGRES_PORT:-5432}"

# RabbitMQ
rabbitmq_url="${RABBITMQ_URL:-amqp://guest:guest@rabbitmq:5672/}"
rabbitmq_host_port="${rabbitmq_url#*@}"
rabbitmq_host_port="${rabbitmq_host_port%/}"
rabbitmq_host="${rabbitmq_host_port%%:*}"
rabbitmq_port="${rabbitmq_host_port##*:}"
wait_for_service "RabbitMQ" "$rabbitmq_host" "$rabbitmq_port"

# Redis
redis_address="${REDIS_ADDRESS:-redis:6379}"
redis_host="${redis_address%%:*}"
redis_port="${redis_address##*:}"
wait_for_service "Redis" "$redis_host" "$redis_port"

# SMTP
smtp_addr="${MAIL_SMTP_ADDR:-smtp4dev:25}"
smtp_host="${smtp_addr%%:*}"
smtp_port="${smtp_addr##*:}"
wait_for_service "SMTP" "$smtp_host" "$smtp_port"

echo "All dependencies are ready; starting application"

exec "$@"
