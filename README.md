# next-deps-srv

Dependency registry server for the Bottles Next project.

Provides a REST API to store, build and serve Wine runtime dependencies.

## Usage

On first run, an `admin` account is created and the token is printed to stdout.

```bash
./next-deps-srv --config config.yaml
```

Set the following environment variables or use a config file:

```yaml
host: 0.0.0.0
port: 8080
storage:
  driver: s3 # or "local"
  bucket: deps-bucket
  endpoint: "http://minio.local:9000"
  region: my-local-region
  access_key: ""
  secret_key: ""
data_dir: ./data
jwt_secret: changeme
```

## Build

### Backend

Requires Go 1.22+, `7z` and `cabextract` on PATH.

```bash
go build -o next-deps-srv ./cmd/server
```

### Frontend

Requires Node 18+.

```bash
cd frontend
npm install
npm run build
```

The compiled assets land in `frontend/dist/` and are embedded into the binary at build time.

## License

MIT
