# GoHarvest

GoHarvest is a lightweight, scalable price-harvesting and processing pipeline built with Go, RabbitMQ, PostgreSQL, Prometheus, and Grafana.

It decouples data ingestion from database persistence using an asynchronous message queue, allowing worker nodes to ingest and record price updates independently without blocking scraping jobs.

---

## Architecture Overview

```
 [ Scraper Service ]
         │
         ▼  (JSON payloads)
 [ RabbitMQ Queue ] (price_items)
         │
         ▼
  [ Worker Service ]
         │
    ┌────┴────────────────────────┐
    ▼                             ▼
 [ PostgreSQL ]         [ Prometheus / Grafana ]
 (items, price_history)   (Metrics & Monitoring)
```

- **Scraper Service (`/scraper`)**: Collects or generates product item data and publishes JSON payloads to RabbitMQ.
- **Worker Service (`/worker`)**: Consumes messages from RabbitMQ, upserts product records into PostgreSQL, and appends time-series price data to the history table.
- **RabbitMQ**: AMQP message broker managing workload distribution between scrapers and workers.
- **PostgreSQL**: Relational storage for item metadata and historical price trends with automatic schema initialization.
- **Prometheus & Grafana**: Monitoring stack to observe message throughput and system status.

---

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/)
- [Go 1.22+](https://go.dev/dl/) (optional, for local development without containers)

### Running with Docker Compose

To start the full environment (RabbitMQ, Postgres, Prometheus, Grafana, Scraper, and Worker):

```bash
docker-compose up --build
```

Once running, the services will be accessible at:

| Service | Address / URL | Description |
|---|---|---|
| RabbitMQ Management | `http://localhost:15672` | AMQP Dashboard (`guest` / `guest`) |
| PostgreSQL | `localhost:5432` | Database (`itemharvester`) |
| Prometheus | `http://localhost:9090` | Metrics Collector |
| Grafana | `http://localhost:3000` | Visualizations & Dashboards |

To stop the services while preserving database volume data:

```bash
docker-compose down
```

To stop services and remove volumes:

```bash
docker-compose down -v
```

---

## Database Schema

The worker service automatically initializes the relational schema on startup if the tables do not exist.

### `users`
Stores user profile accounts.

```sql
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `items`
Stores product records associated with each user. Unique per user and URL.

```sql
CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_user_url UNIQUE (user_id, url)
);
```

### `price_history`
Stores point-in-time price observations linked to each item.

```sql
CREATE TABLE IF NOT EXISTS price_history (
    id SERIAL PRIMARY KEY,
    item_id INT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    price NUMERIC(10, 2) NOT NULL,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## Configuration

Services can be configured via environment variables set in `docker-compose.yml`:

| Variable | Default Value | Description |
|---|---|---|
| `DB_HOST` | `postgres` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `user` | PostgreSQL username |
| `DB_PASSWORD` | `password` | PostgreSQL password |
| `DB_NAME` | `itemharvester` | Database name |

---

## Project Structure

```
.
├── scraper/              # Scraper microservice source code
│   └── scraper.go
├── worker/               # Worker microservice source code
│   └── worker.go
├── Dockerfile.scraper    # Dockerfile for scraper service
├── Dockerfile.worker     # Dockerfile for worker service
├── docker-compose.yml    # Multi-container orchestration specification
├── prometheus.yml        # Prometheus collection configuration
├── go.mod                # Go module dependencies
└── todo.txt              # Project roadmap & pending tasks
```

---

## License

MIT
