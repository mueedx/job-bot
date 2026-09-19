# Quickstart: Control Center

## Run

```bash
# Terminal 1
cd backend && go run ./cmd/server

# Terminal 2
cd frontend && npm install && npm run dev
```

- API: http://localhost:8000/docs  
- UI: http://localhost:3000  

## Seed

```bash
curl -X POST http://localhost:8000/api/dev/seed
```

## Verify

1. Board shows five columns with seeded cards.
2. Open a High Match role — JD, score, cover letter visible.
3. Edit cover letter → Save → reload → persisted.
4. Switch track → persists.
5. Discard → card moves to Archived.
6. Approve & Submit → clear not-ready/501 message (no false “sent”).
7. `/stats` shows counts + daily limit.
