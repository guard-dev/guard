#!/bin/bash

# Ensure required environment variables are set
: "${GEMINI_SECRET_KEY:?GEMINI_SECRET_KEY is not set}"
: "${AWS_REGION:?AWS_REGION is not set}"
: "${AWS_ACCESS_KEY_ID:?AWS_ACCESS_KEY_ID is not set}"
: "${AWS_SECRET_ACCESS_KEY:?AWS_SECRET_ACCESS_KEY is not set}"

# Start Supervisor to manage services
exec supervisord -c /etc/supervisor/supervisord.conf
