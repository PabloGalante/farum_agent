# Farum – A Multi-Agent Mental Health Companion (Go + GCP-Ready)

Farum is a **multi-agent psychological companion** designed to provide emotional support, structured guidance, and long-term reflection.  
It uses a **sequential agent pipeline**, **tools**, and **session memory** to create a rich, evolving conversation with users.

This project was developed as part of the Kaggle Generative AI Capstone and is implemented entirely in **Go**, with a clean hexagonal architecture and optional integration with **Google Cloud Platform** (Vertex AI + Firestore + Cloud Run).

---

## 🌱 Project Goals

Farum aims to:

- Listen empathetically to users  
- Clarify their concerns  
- Propose small, realistic action plans  
- Close with reflective messages  
- Maintain a long-term **journal** of user interactions  

It operates fully **locally** using a lightweight mock LLM (free and safe) but can be configured to run on **GCP** with Vertex AI and Firestore.

---

## ✨ Key Features

### **🧠 Multi-Agent System**

Farum processes each user message through three sequential agents:

1. **ListenerAgent**  
   Listens, empathizes, paraphrases the concern.

2. **PlannerAgent**  
   Produces a short action plan (2–4 concrete steps).

3. **ReflectorAgent**  
   Provides a reflective closing message and triggers journaling.

### **🔧 Tools (Custom Tool Integration)**

Farum implements a generic `Tool` interface and includes:

- **JournalTool**: writes structured `JournalEntry` objects into the journal store.

### **📚 Memory (Short-term + Long-term)**

- **Short-term**: Session messages + context passed to agents.
- **Long-term**: Persistent `JournalEntry` list (in-memory or Firestore).

### **📡 HTTP API**

REST interface:

- `POST /sessions`
- `GET /sessions/{id}`
- `POST /sessions/{id}/messages`
- `GET /users/{user_id}/journal?limit=N`
- `GET /healthz`

### **🔍 Observability**

- Structured logging (`slog`)
- Per-agent timing
- Structured fields: session_id, user_id, mode, agent, etc.

### **☁️ Cloud-Ready**

- Vertex AI (Gemini) LLM Support  
- Firestore Store (sessions/messages implemented; journal WIP)  
- Cloud Run deployment model  

---

## 🏗️ Architecture Overview

### Hexagonal (Ports & Adapters)

```plain
/cmd/farum-api
/internal
  /adapters
    /http         → REST API
    /llm          → Mock LLM + Vertex clientg
    /storage
      /memory     → in-memory stores
      /firestore  → Firestore store
  /app
    /conversation → Session & message orchestration
    /agentflow    → Multi-agent pipeline (Listener, Planner, Reflector)
    /journal      → Journal read service
    /tools
      tools.go
      journal_tool.go
  /domain
    entities: Session, Message, ConversationContext
    memory: JournalEntry, JournalAction
    interfaces: LLMClient, SessionStore, MessageStore, JournalStore
  /observability → logger & helpers
  /config         → env-based configuration
```

---

## 🚀 Running Locally (Recommended)

Local mode is **free**, **safe**, and requires **no cloud credentials**.

### 1. Clone the repo

```bash
git clone <your-repo-url>
cd farum-agent
```

### 2. Set environment variables (optional)

```bash
export FARUM_MODE=local
export FARUM_STORAGE_BACKEND=memory
export FARUM_USE_MOCK_LLM=true
export FARUM_PORT=8080
```

### 3. Run the API

```bash
go run ./cmd/farum-api
```

You should see logs like:

```plain
{"msg":"starting Farum","mode":"local","use_mock_llm":true}
{"msg":"[STORE] Using in-memory storage"}
```

---

## 🧪 Testing the API

### Create a session

```bash
curl -X POST http://localhost:8080/sessions \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test-user","preferred_mode":"check_in","title":"First session"}'
```

### Send a message

```bash
curl -X POST http://localhost:8080/sessions/<SESSION_ID>/messages \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test-user","text":"I feel anxious today"}'
```

### Read the journal

```bash
curl "http://localhost:8080/users/test-user/journal?limit=10"
```

---

## 🧩 Configuration Reference

Environment variables:

| Variable | Description | Default |
|---------|-------------|---------|
| `FARUM_MODE` | `local` or `gcp` | `local` |
| `FARUM_STORAGE_BACKEND` | `memory` or `firestore` | `memory` |
| `FARUM_USE_MOCK_LLM` | Use mock model | `true` |
| `FARUM_PORT` | HTTP port | `8080` |
| `FARUM_GCP_PROJECT` | GCP project (for Firestore/Vertex) | _required for GCP_ |
| `FARUM_GCP_LOCATION` | GCP region | `"us-central1"` |
| `FARUM_MODEL_NAME` | Vertex model | `"gemini-2.5-flash"` |

---

## ☁️ Running on GCP (Design Overview)

Farum is fully designed to run in production on GCP:

### Services Used

- **Vertex AI** → LLM backend  
- **Firestore** → Session/message storage  
- **Cloud Run** → Serverless deployment  
- **Cloud Logging** → Observability  
- **Secret Manager** → API keys (optional)

### Steps (high-level)

1. Create a GCP project  
2. Enable Vertex AI + Firestore  
3. Build container:

   ```bash
   gcloud builds submit --tag gcr.io/$PROJECT_ID/farum
   ```

4. Deploy to Cloud Run:

   ```bash
   gcloud run deploy farum \
     --image gcr.io/$PROJECT_ID/farum \
     --region $REGION
   ```

5. Set environment variables accordingly.

Firestore support is implemented for session/messages.  
Journal persistence is ready to implement (interfaces already designed).

---

## 🎓 Kaggle Capstone: How Farum Meets Requirements

Farum implements **all the required capstone concepts**:

### ✔ Multi-Agent System

- ListenerAgent  
- PlannerAgent  
- ReflectorAgent  
- Sequential orchestration via `Orchestrator`

### ✔ Tools

- Generic `Tool` interface  
- Custom `JournalTool`  

### ✔ Sessions & Memory

- Session history (`SessionStore`, `MessageStore`)  
- Long-term journaling (`JournalEntry`, `JournalStore`)  

### ✔ Observability

- Structured logs (`slog`)  
- Per-agent metrics  
- Context-aware logging  

### ✔ Deployment-Ready

- Local mode (for safe testing)  
- GCP mode using Vertex AI and Firestore  
- Cloud-Run-ready server

---

## 📄 License

MIT License.

---

## ✉ Contact

If you have questions or want to collaborate, feel free to reach out!
