# RoCode Web UI

A modern web interface for RoCode, providing a browser-based alternative to the terminal UI.

## Features

- Real-time chat interface with streaming responses
- Provider and model selection
- Tool execution visualization
- Session management (create, delete, share)
- File upload support
- Syntax highlighting for code blocks
- Dark/light theme support
- Cost tracking display

## Development

### Prerequisites

- RoCode server running on port 4096
- Node.js 18+ or Bun
- Modern web browser

### Setup

```bash
# Install dependencies
cd packages/web-ui
bun install

# Start development server
bun run dev
```

The web UI will be available at http://localhost:3000

### Running with RoCode Server

Make sure the RoCode server is running first:

```bash
# In one terminal, start the RoCode server
cd packages/opencode
bun run dev

# In another terminal, start the web UI
cd packages/web-ui
bun run dev
```

### Build

```bash
bun run build
```

The production build will be in the `dist` directory.

## Architecture

The web UI communicates with the RoCode server through:
- REST API endpoints for synchronous operations
- Server-Sent Events (SSE) for real-time updates

### Key Technologies

- **React** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool and dev server
- **Tailwind CSS** - Styling
- **Zustand** - State management
- **Monaco Editor** - Code editing
- **React Markdown** - Markdown rendering

### Project Structure

```
src/
├── api/          # API client and SSE handling
├── components/   # React components
│   ├── chat/     # Chat-related components
│   ├── provider/ # Provider selection
│   └── session/  # Session management
├── store/        # Zustand state stores
├── styles/       # Global styles
└── types/        # TypeScript type definitions
```

## Contributing

When adding new features:
1. Ensure TypeScript types are properly defined
2. Follow the existing component structure
3. Use Tailwind CSS for styling
4. Test with different providers and models
5. Ensure SSE events are properly handled
