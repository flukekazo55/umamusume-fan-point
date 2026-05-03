# Uma Musume Fan Point Tracker

Mobile-first web app converted from `source.xlsx`.

## Run Locally

Start the backend:

```powershell
cd backend
go run ./cmd/server
```

Start the frontend:

```powershell
cd frontend
npm install
npm start
```

Open `http://localhost:4200`.

## Data Source

The Go backend reads `source.xlsx` from the repo root by default. Override it with:

```powershell
$env:DATA_FILE="C:\path\to\file.xlsx"
go run ./cmd/server
```

To persist and edit player points, run the backend with MongoDB:

```powershell
$env:MONGO_URI="mongodb://localhost:27017"
$env:MONGO_DATABASE="umamusume_fan_point"
go run ./cmd/server
```

When MongoDB is enabled, the backend seeds `months` and `players` from `source.xlsx` only if the Mongo database is empty. After that, reads and player edits come from MongoDB.

API endpoints:

- `GET /api/health`
- `GET /api/months`
- `POST /api/months`
- `GET /api/months/{id}`
- `PUT /api/months/{id}`
- `DELETE /api/months/{id}`
- `GET /api/months/{monthId}/players`
- `POST /api/months/{monthId}/players`
- `GET /api/months/{monthId}/players/{playerName}`
- `PUT /api/months/{monthId}/players/{playerName}`
- `DELETE /api/months/{monthId}/players/{playerName}`

Month create/update body:

```json
{
  "id": "Jun2026",
  "label": "June 2026",
  "startDate": "2026-06-01",
  "endDate": "2026-06-30",
  "dates": ["2026-06-01", "2026-06-07", "2026-06-14", "2026-06-21", "2026-06-30"],
  "sourceMonthId": "May2026"
}
```

`sourceMonthId` is used only when creating a month. It clones players from the old month into the new month with their latest fan total as the first snapshot, so the new month starts with the same standard player list.

Player create/update body:

```json
{
  "name": "PlayerName",
  "debt": 0,
  "note": "",
  "snapshots": [
    { "date": "2026-05-01", "fans": 1000000 },
    { "date": "2026-05-10", "fans": 1500000 }
  ]
}
```
