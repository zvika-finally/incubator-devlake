# Squadcast Plugin for Apache DevLake

This plugin enables ingestion of incidents from Squadcast into DevLake for unified engineering analytics.

## Features
- Collects incidents from Squadcast API
- Maps Squadcast data to DevLake domain models
- Extensible for alerts, users, schedules, etc.

## Setup
- Configure a Squadcast connection via the DevLake API or UI.
- Add the plugin to your pipeline blueprint.

## Connection Configuration
- Configure your Squadcast API key via the DevLake UI or API.
- The plugin will use this key to fetch incidents.

## Roadmap
- [x] Data model and migration
- [x] Collector and extractor
- [x] Plugin registration
- [x] Connection API
- [x] E2E test scaffolding
- [ ] UI integration 