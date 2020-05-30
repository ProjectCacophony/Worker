#!/usr/bin/env bash

# should have the following environment variables set:
# PORT
# HASH
# ENVIRONMENT
# LOGGING_DISCORD_WEBHOOK
# DISCORD_TOKENS
# DOCKER_IMAGE_HASH
# DB_DSN
# REDIS_ADDRESS
# REDIS_PASSWORD
# CLUSTER_ENVIRONMENT
# FEATUREFLAG_UNLEASH_URL
# FEATUREFLAG_UNLEASH_INSTANCE_ID
# ERRORTRACKING_RAVEN_DSN
# POLR_BASE_URL
# POLR_API_KEY
# AMQP_DSN
# IEXCLOUD_API_SECRET
# DISCORD_API_BASE
# PATREON_CAMPAIGN_ID
# PATREON_CREATORS_ACCESS_TOKEN
# WEVERSE_TOKEN
# TIKTOK_API_BASE_URL

template="k8s/manifest.tmpl.yaml"
target="k8s/manifest.yaml"

cp "$template" "$target"
sed -i -e "s|{{PORT}}|$PORT|g" "$target"
sed -i -e "s|{{HASH}}|$HASH|g" "$target"
sed -i -e "s|{{ENVIRONMENT}}|$ENVIRONMENT|g" "$target"
sed -i -e "s|{{LOGGING_DISCORD_WEBHOOK}}|$LOGGING_DISCORD_WEBHOOK|g" "$target"
sed -i -e "s|{{DISCORD_TOKENS}}|$DISCORD_TOKENS|g" "$target"
sed -i -e "s|{{DOCKER_IMAGE_HASH}}|$DOCKER_IMAGE_HASH|g" "$target"
sed -i -e "s|{{DB_DSN}}|$DB_DSN|g" "$target"
sed -i -e "s|{{REDIS_ADDRESS}}|$REDIS_ADDRESS|g" "$target"
sed -i -e "s|{{REDIS_PASSWORD}}|$REDIS_PASSWORD|g" "$target"
sed -i -e "s|{{CLUSTER_ENVIRONMENT}}|$CLUSTER_ENVIRONMENT|g" "$target"
sed -i -e "s|{{FEATUREFLAG_UNLEASH_URL}}|$FEATUREFLAG_UNLEASH_URL|g" "$target"
sed -i -e "s|{{FEATUREFLAG_UNLEASH_INSTANCE_ID}}|$FEATUREFLAG_UNLEASH_INSTANCE_ID|g" "$target"
sed -i -e "s|{{ERRORTRACKING_RAVEN_DSN}}|$ERRORTRACKING_RAVEN_DSN|g" "$target"
sed -i -e "s|{{INSTAGRAM_SESSION_IDS}}|$INSTAGRAM_SESSION_IDS|g" "$target"
sed -i -e "s|{{POLR_BASE_URL}}|$POLR_BASE_URL|g" "$target"
sed -i -e "s|{{POLR_API_KEY}}|$POLR_API_KEY|g" "$target"
sed -i -e "s|{{AMQP_DSN}}|$AMQP_DSN|g" "$target"
sed -i -e "s|{{IEXCLOUD_API_SECRET}}|$IEXCLOUD_API_SECRET|g" "$target"
sed -i -e "s|{{DISCORD_API_BASE}}|$DISCORD_API_BASE|g" "$target"
sed -i -e "s|{{PATREON_CAMPAIGN_ID}}|$PATREON_CAMPAIGN_ID|g" "$target"
sed -i -e "s|{{PATREON_CREATORS_ACCESS_TOKEN}}|$PATREON_CREATORS_ACCESS_TOKEN|g" "$target"
sed -i -e "s|{{WEVERSE_TOKEN}}|$WEVERSE_TOKEN|g" "$target"
sed -i -e "s|{{TIKTOK_API_BASE_URL}}|$TIKTOK_API_BASE_URL|g" "$target"
