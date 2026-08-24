# OrbitLink

OrbitLink coordinates satellite pass reception, antenna ownership, RF plans, receiver acquisition and durable frame recording for a ground station.

Run locally with Go 1.26.2:

```text
go run ./cmd/orbitlink -listen 127.0.0.1:19705 -state-root ./var/orbitlink
```

Open `/passes`, `/antennas`, `/rf-chains` and `/recordings` for the four operations views. The service exposes `/healthz` and matching JSON APIs under `/api`.
