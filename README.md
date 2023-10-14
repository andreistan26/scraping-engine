# Scraping Engine

## Architecture Overview

```mermaid
sequenceDiagram
Cron Job->>Database: Scan for due jobs
Cron Job->>RabbitMQ: Push due jobs to RMQ
RabbitMQ->>Worker1..N: Pop queued scraping jobs
Worker1..N->>Database: Update scraped items
Cron Job->>Cron Job: Repeat
```
