# Guardian Tracker

A web-based app tailored for Destiny 2 players that integrates Bungie APIs to analyze users' collections and identify missing items with prioritized acquisition plans.

## 🌟 Core Functionality

- **OAuth Integration**: Secure login with Bungie accounts
- **Collection Analysis**: Automatically identify missing items and classify by difficulty
- **Wish List Management**: Track and prioritize desired items
- **Weekly Reset Notifications**: Alerts for easily obtainable or high-value items
- **Flexible Frontend Queries**: GraphQL for efficient data fetching

## 🛠️ Architecture

### Frontend

- **Framework**: React + TypeScript
- **GraphQL Client**: Apollo Client
- **UI Toolkit**: Tailwind CSS + shadcn/ui

### Backend

- **GraphQL Layer**: Apollo Server
- **REST Microservices**: Go + Gin
- **Databases**: PostgreSQL, MongoDB/DynamoDB, Redis

### Infrastructure

- **Deployment**: AWS EKS (Kubernetes)
- **CI/CD**: GitHub Actions
- **IaC**: Terraform
- **Additional**: AWS Lambda, SNS, S3

## 📁 Project Structure

```text
guardian-tracker/
├── frontend/                 # React + TypeScript frontend
├── backend/
│   ├── graphql-service/     # Apollo Server (Node.js/TypeScript)
│   ├── auth-service/        # Go authentication microservice
│   └── bungie-service/      # Go Bungie API integration service
├── database/                 # Database initialization scripts
├── k8s/                      # Kubernetes manifests
└── .github/                  # CI/CD workflows (coming soon)
```

## 🚀 Getting Started

### Prerequisites

- Node.js 18+
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL
- MongoDB or DynamoDB
- Redis

### Quick Start

1. **Clone the repository**

   ```bash
   git clone <repo-url>
   cd guardian-tracker
   ```

2. **Set up environment variables**

   **Important:** Before running any services, you must configure environment variables.

   ```bash
   # Copy environment templates
   cp .env.example .env
   cd backend/auth-service && cp .env.example .env && cd ../..
   cd backend/bungie-service && cp .env.example .env && cd ../..
   cd backend/graphql-service && cp .env.example .env && cd ../..
   cd frontend && cp .env.example .env.local && cd ..
   ```

   **Next:** Edit each `.env` file with your actual credentials.

   📖 **See [ENVIRONMENT_SETUP.md](./ENVIRONMENT_SETUP.md) for detailed configuration instructions.**

3. **Get Bungie API Credentials**

   You'll need to create a Bungie.net application to get:

   - API Key
   - OAuth Client ID
   - OAuth Client Secret

   📖 **See [ENVIRONMENT_SETUP.md](./ENVIRONMENT_SETUP.md#-getting-bungie-api-credentials) for step-by-step instructions.**

4. **Install dependencies**

   ```bash
   # Frontend
   cd frontend && npm install && cd ..

   # GraphQL Service
   cd backend/graphql-service && npm install && cd ../..

   # Go services (download dependencies)
   cd backend/auth-service && go mod tidy && cd ../..
   cd backend/bungie-service && go mod tidy && cd ../..
   ```

5. **Start services**

   **Option A: Using Docker Compose (Recommended)**

   ```bash
   docker-compose up -d
   ```

   **Option B: Run services individually**

   ```bash
   # Terminal 1 - Auth Service
   cd backend/auth-service
   go run main.go

   # Terminal 2 - Bungie Service
   cd backend/bungie-service
   go run main.go

   # Terminal 3 - GraphQL Service
   cd backend/graphql-service
   npm run dev

   # Terminal 4 - Frontend
   cd frontend
   npm start
   ```

6. **Access the application**
   - Frontend: <http://localhost:3000>
   - GraphQL Playground: <http://localhost:4000/graphql>
   - Auth Service: <http://localhost:8081>
   - Bungie Service: <http://localhost:8082>

### Development with Kubernetes

For Kubernetes deployment on Minikube:

```bash
cd k8s
./startup.bat  # Windows
# or
./startup.sh   # Linux/Mac
```

See [k8s/README.md](./k8s/README.md) for detailed Kubernetes setup instructions.

## 📚 Documentation

- [Environment Setup Guide](./ENVIRONMENT_SETUP.md) ⭐ **Start here!**
- [Frontend Setup](./frontend/README.md)
- [Backend Services](./backend/README.md)
- [Kubernetes Deployment](./k8s/README.md)
- [API Documentation](./docs/api.md) (Coming soon)

## 🔧 Development

This project uses a microservices architecture with:

- REST APIs for core services (Go)
- GraphQL aggregation layer (Node.js)
- React frontend with Apollo Client
- Containerized deployment on Kubernetes

## 📄 License

MIT License - see [LICENSE](./LICENSE) file for details.
