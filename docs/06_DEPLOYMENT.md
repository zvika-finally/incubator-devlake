# Deployment Guide

## Deploying to Render

- See `infra/render/README.md` for step-by-step instructions.
- Render manages PostgreSQL and Grafana, reducing operational overhead.
- Use the provided `infra/render/render.yaml` blueprint for a one-click deployment.
- For local development, use `docker-compose-dev.yml`.
- For production, ensure secrets are set in the Render dashboard and not committed to code. 