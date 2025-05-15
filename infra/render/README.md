# Deploying Apache DevLake to Render

This guide explains how to deploy DevLake (backend, config UI, PostgreSQL, and Grafana) to Render using the provided blueprint.

## Steps

1. **Create a new Render Blueprint deployment:**
   - Go to [Render.com](https://render.com/)
   - Create a new Blueprint deployment and upload `infra/render/render.yaml`

2. **Services Created:**
   - `devlake-backend` (API, port 8080)
   - `devlake-config-ui` (UI, port 4000)
   - `devlake-db` (PostgreSQL, managed)
   - `devlake-grafana` (Grafana, managed)

3. **Environment Variables:**
   - Sensitive values (e.g., secrets) should be set in the Render dashboard, not in code.
   - The backend and frontend services are pre-configured to use the managed database and Grafana.

4. **Accessing Services:**
   - After deployment, Render will provide public URLs for each service.
   - Update `DEVLAKE_ENDPOINT` and `GRAFANA_ENDPOINT` in the config UI service if the URLs differ from the defaults.

5. **Health Checks:**
   - Backend: `/health`
   - Frontend: `/`

6. **Troubleshooting:**
   - Check service logs in the Render dashboard.
   - Ensure all environment variables are set.
   - For database connection issues, verify the `DATABASE_URL` is correct.

7. **Next Steps:**
   - Log in to the Config UI and connect your data sources.
   - Use Grafana for dashboards and metrics. 