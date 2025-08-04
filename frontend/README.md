# Guardian Tracker Frontend

A React + TypeScript frontend for the Guardian Tracker application, built with modern UI components and GraphQL integration.

## 🛠️ Tech Stack

- **Framework**: React 18 + TypeScript
- **Styling**: Tailwind CSS + shadcn/ui components
- **GraphQL**: Apollo Client
- **Routing**: React Router
- **State Management**: Apollo Client cache + React hooks
- **Build Tool**: Create React App
- **Development**: Hot reload with Docker

## 🚀 Getting Started

### Prerequisites

- Node.js 18+
- npm or yarn

### Local Development

1. **Install dependencies**
   ```bash
   npm install
   ```

2. **Set up environment variables**
   ```bash
   cp ../.env.example .env.local
   # Edit .env.local with your configuration
   ```

3. **Start development server**
   ```bash
   npm start
   ```

4. **Open in browser**
   ```
   http://localhost:3000
   ```

### Docker Development

```bash
# From project root
docker-compose up frontend
```

## 📁 Project Structure

```
src/
├── components/          # Reusable UI components
│   ├── ui/             # Basic UI components (Button, Card, etc.)
│   └── Navigation.tsx  # App navigation
├── pages/              # Route components
│   ├── Dashboard.tsx   # Main dashboard
│   ├── Collections.tsx # Collections view
│   ├── WishList.tsx    # Wish list management
│   └── Login.tsx       # Authentication
├── graphql/            # GraphQL queries and mutations
│   ├── queries.ts      # GraphQL queries
│   └── mutations.ts    # GraphQL mutations
├── types/              # TypeScript type definitions
├── lib/                # Utility functions
└── App.tsx             # Main app component
```

## 🎨 UI Components

The app uses a custom design system built on top of Tailwind CSS with Destiny 2-themed colors:

- **Destiny Colors**: Exotic (gold), Legendary (purple), Rare (blue), etc.
- **Responsive Design**: Mobile-first approach
- **Dark Theme**: Custom dark theme optimized for gaming
- **Animations**: Smooth transitions and hover effects

## 🔌 GraphQL Integration

Apollo Client provides:
- **Caching**: Intelligent query caching and updates
- **Optimistic UI**: Immediate UI updates for better UX
- **Error Handling**: Comprehensive error boundaries
- **Type Safety**: Generated TypeScript types from schema

## 🏗️ Key Features

### Dashboard
- Collection progress overview
- Weekly vendor recommendations
- Featured activities with rewards
- Quick action buttons

### Collections
- Browse missing items by category (weapons, armor, exotics)
- Filter by acquisition difficulty
- Visual indicators for rarity and sources
- Add items to wish list

### Wish List
- Prioritize desired items
- Add personal notes
- Track acquisition progress
- Sort by priority and date

### Authentication
- Secure Bungie OAuth integration
- Token management
- Profile display
- Logout functionality

## 🧪 Available Scripts

```bash
# Development
npm start              # Start development server
npm test              # Run tests
npm run build         # Build for production

# Code Quality
npm run lint          # Run ESLint
npm run lint:fix      # Fix ESLint issues
npm run type-check    # TypeScript type checking
```

## 🔧 Configuration

### Environment Variables

```env
REACT_APP_GRAPHQL_ENDPOINT=http://localhost:4000/graphql
REACT_APP_AUTH_ENDPOINT=http://localhost:8081
```

### Apollo Client Configuration

- **URI**: GraphQL endpoint from environment
- **Auth Link**: Automatic JWT token injection
- **Cache**: Custom type policies for optimal caching
- **Error Policy**: Graceful error handling

## 🎯 Performance Optimizations

- **Code Splitting**: Lazy loading of route components
- **Image Optimization**: Automatic image URL validation
- **Debounced Search**: Optimized search input handling
- **Memoization**: React.memo and useMemo for expensive operations

## 🔐 Security

- **JWT Storage**: Secure token storage in localStorage
- **XSS Protection**: Sanitized user inputs
- **HTTPS**: Production deployment over HTTPS
- **CSP**: Content Security Policy headers

## 🚀 Deployment

### Production Build

```bash
npm run build
```

### Docker Production

```dockerfile
FROM nginx:alpine
COPY build/ /usr/share/nginx/html/
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
```

## 🤝 Contributing

1. Follow the existing code style
2. Add TypeScript types for new features
3. Include responsive design considerations
4. Test on mobile devices
5. Update documentation for new components

## 📊 Monitoring

The frontend includes:
- Error boundaries for crash recovery
- Performance monitoring hooks
- User action analytics
- GraphQL query performance tracking
