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

```
guardian-tracker/
├── frontend/                 # React + TypeScript frontend
├── backend/
│   ├── graphql-service/     # Apollo Server (Node.js/TypeScript)
│   ├── auth-service/        # Go authentication microservice
│   └── bungie-service/      # Go Bungie API integration service
├── infrastructure/
│   ├── terraform/           # Infrastructure as Code
│   └── k8s/                # Kubernetes manifests
├── docker/                  # Docker configurations
└── .github/                # CI/CD workflows
```

## 🚀 Getting Started

1. **Prerequisites**

   - Node.js 18+
   - Go 1.21+
   - Docker & Docker Compose
   - PostgreSQL
   - MongoDB or DynamoDB
   - Redis

2. **Setup Development Environment**

   ```bash
   # Clone and setup
   git clone <repo-url>
   cd guardian-tracker

   # Install frontend dependencies
   cd frontend && npm install

   # Build Go services
   cd ../backend/auth-service && go mod tidy
   cd ../bungie-service && go mod tidy
   cd ../graphql-service && npm install

   # Start development environment
   docker-compose up -d
   ```

## 📚 Documentation

- [Frontend Setup](./frontend/README.md)
- [Backend Services](./backend/README.md)
- [Infrastructure](./infrastructure/README.md)
- [API Documentation](./docs/api.md)

## 🔧 Development

This project uses a microservices architecture with:

- REST APIs for core services (Go)
- GraphQL aggregation layer (Node.js)
- React frontend with Apollo Client
- Containerized deployment on Kubernetes

## 📄 License

MIT License - see [LICENSE](./LICENSE) file for details.
