# GitHub Migrator - Frontend

Modern React frontend for the GitHub Migrator built with Vite, TypeScript, and Tailwind CSS.

## Tech Stack

- **React 18** with TypeScript
- **Vite** - Fast build tooling
- **Tailwind CSS** - Utility-first styling
- **React Router** - Client-side routing
- **Recharts** - Data visualization
- **Axios** - HTTP client

## Getting Started

### Install Dependencies

```bash
npm install
```

### Development

```bash
npm run dev
```

The application will be available at `http://localhost:3000`.

The dev server is configured to proxy API requests to `http://localhost:8080`.

### Build for Production

```bash
npm run build
```

The production build will be in the `dist/` directory.

### Preview Production Build

```bash
npm run preview
```

### Linting

```bash
npm run lint
```

### Type Checking

```bash
npm run type-check
```

## Project Structure

```
src/
├── components/          # React components
│   ├── common/         # Shared components (Navigation, Badge, etc.)
│   ├── Dashboard/      # Repository dashboard
│   ├── RepositoryDetail/ # Repository detail view
│   ├── Analytics/      # Analytics dashboard
│   ├── BatchManagement/ # Batch management UI
│   └── SelfService/    # Self-service migration
├── services/           # API services
│   └── api.ts         # API client
├── types/             # TypeScript types
│   └── index.ts       # Type definitions
├── utils/             # Utility functions
│   └── format.ts      # Formatting helpers
├── App.tsx            # Main application
├── main.tsx           # Entry point
└── index.css          # Global styles

## Features

### Dashboard
- Repository grid view with filtering
- Real-time status updates (10s polling)
- Search functionality
- Quick repository overview cards

### Repository Detail
- Complete repository profile
- Migration controls (Dry Run, Start Migration, Retry)
- Migration history timeline
- Detailed logs with filtering

### Analytics
- Migration progress overview
- Status breakdown charts (bar and pie)
- Average migration time
- Detailed status table

### Batch Management
- View and manage migration batches
- Start batch migrations
- View repositories in batch
- Real-time status updates

### Self-Service
- Developer-friendly migration interface
- Bulk repository input
- Dry run support
- Success/error feedback

## API Integration

The frontend communicates with the backend API at `/api/v1`. All requests are proxied through Vite's dev server.

Key endpoints:
- `GET /api/v1/repositories` - List repositories
- `GET /api/v1/repositories/:fullName` - Get repository details
- `POST /api/v1/migrations/start` - Start migration
- `GET /api/v1/batches` - List batches
- `GET /api/v1/analytics/summary` - Get analytics

## Development Notes

### Real-time Updates
Components use polling intervals to fetch updates:
- Dashboard: 10 seconds
- Repository Detail: 10 seconds  
- Analytics: 30 seconds
- Batch Management: 15 seconds

### Styling
- Tailwind CSS for all styling
- Custom theme with blue primary colors
- Responsive design (mobile, tablet, desktop)
- Minimal, Apple-like aesthetic

### Type Safety
All components are fully typed with TypeScript. API responses match backend models.

## Browser Support

- Modern browsers (Chrome, Firefox, Safari, Edge)
- ES2020+ features
- No IE11 support

