# IPTV Panel - Vue Element Plus Admin Integration

## Overview
IPTV management panel built with Vue 3 + TypeScript using the vue-element-plus-admin professional template.

## Technology Stack
- **Frontend**: Vue 3 (Composition API), TypeScript, Vite
- **UI Framework**: Element Plus
- **State Management**: Pinia
- **Router**: Vue Router 4
- **HTTP Client**: Axios with useAxios hook
- **CSS**: UnoCSS (Utility-first)
- **Backend**: Go server on port 8080
- **Dev Server**: Port 4000

## Project Structure
```
pannel/
├── src/
│   ├── views/IPTV/          # IPTV management views
│   │   ├── Dashboard.vue     # Main dashboard with stats & monitoring
│   │   ├── Playlists.vue     # Playlist management
│   │   ├── Channels.vue      # Channel management
│   │   ├── Users.vue         # User management
│   │   ├── Relays.vue        # Stream relay management
│   │   ├── GeneratePlaylist.vue  # Custom playlist generator
│   │   └── ImportM3U.vue     # M3U import interface
│   ├── router/
│   │   └── index.ts          # IPTV-only routes
│   ├── api/login/
│   │   └── index.ts          # Backend API integration
│   └── store/
│       └── modules/locale.ts # Default language: English
```

## Features

### 1. Dashboard
- Real-time statistics (playlists, channels, relays count)
- Bandwidth monitoring
- Recent channels list
- API: `/api/stats`

### 2. Playlists Management
- View all playlists with channel counts
- Delete playlists with confirmation
- See creation dates and URLs
- API: `/api/playlists` (GET, DELETE)

### 3. Channels Management
- Search channels by name/category
- Filter by category
- Enable/disable channels
- Batch delete functionality
- Channel logos display
- APIs: `/api/channels/search`, `/api/channels/{id}/toggle`, `/api/channels/{id}` (DELETE)

### 4. Users Management
- Create users with username, password, expiry date
- Extend subscriptions
- Reset passwords
- Delete users
- Expiry status indicators (active, warning, expired)
- APIs: `/api/users` (GET, POST, DELETE), `/api/users/{id}/extend`, `/api/users/{id}/reset-password`

### 5. Relays Management
- Create stream relays with source URL
- Copy relay URLs to clipboard
- View active/inactive status
- Delete relays
- APIs: `/api/relays` (GET, POST, DELETE)

### 6. Generate Playlist
- Select user from dropdown
- Choose channels using transfer component
- Generate custom user playlists
- Auto-copy playlist URL to clipboard
- API: `/api/generate-playlist`

### 7. Import M3U
- Import M3U/M3U8 playlists via URL
- Support for EXTINF format
- Category grouping support
- API: `/api/playlists/import`

## Configuration

### Router Configuration
- Clean IPTV-only routes (demo components removed)
- All routes follow template's Layout pattern
- Icons from Element Plus and Iconify

### Authentication
- Login endpoint: `/api/auth/login`
- Logout endpoint: `/api/auth/logout`
- Session-based authentication (cookies)
- Default credentials: `admin` / `admin123`

### API Proxy
Configured in `vite.config.ts`:
```typescript
proxy: {
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true
  },
  '/stream': {
    target: 'http://localhost:8080',
    changeOrigin: true
  }
}
```

## Development

### Install Dependencies
```bash
cd pannel
npm install --legacy-peer-deps
```

### Run Development Server
```bash
npm run dev
```
Access at: `http://localhost:4000`

### Build for Production
```bash
npm run build:pro
```

## Design Patterns Used

### ContentWrap Component
All views wrapped with `ContentWrap` for consistent layout:
```vue
<ContentWrap title="Page Title" message="Description">
  <!-- Content here -->
</ContentWrap>
```

### useAxios Hook
Used for all API calls instead of direct axios:
```typescript
const request = useAxios()
const data = await request.get({ url: '/api/endpoint' })
```

### TypeScript Typing
All components use TypeScript with proper typing:
```typescript
const users = ref<User[]>([])
const loading = ref<boolean>(false)
```

## API Endpoints Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/login` | Admin login |
| POST | `/api/auth/logout` | Admin logout |
| GET | `/api/stats` | Dashboard statistics |
| GET | `/api/playlists` | List all playlists |
| POST | `/api/playlists/import` | Import M3U playlist |
| DELETE | `/api/playlists/{id}` | Delete playlist |
| GET | `/api/channels/search?q=` | Search channels |
| POST | `/api/channels/{id}/toggle` | Enable/disable channel |
| DELETE | `/api/channels/{id}` | Delete channel |
| POST | `/api/channels/batch-delete` | Delete multiple channels |
| GET | `/api/users` | List all users |
| POST | `/api/users` | Create user |
| DELETE | `/api/users/{id}` | Delete user |
| POST | `/api/users/{id}/extend` | Extend subscription |
| POST | `/api/users/{id}/reset-password` | Reset password |
| GET | `/api/relays` | List relays |
| POST | `/api/relays` | Create relay |
| DELETE | `/api/relays/{id}` | Delete relay |
| POST | `/api/generate-playlist` | Generate user playlist |

## Customization

### Adding New Routes
Edit `/home/dindin/mqltv/pannel/src/router/index.ts`:
```typescript
{
  path: '/new-feature',
  component: Layout,
  name: 'NewFeature',
  meta: {},
  children: [
    {
      path: 'index',
      component: () => import('@/views/IPTV/NewFeature.vue'),
      name: 'NewFeatureManagement',
      meta: {
        title: 'New Feature',
        icon: 'vi-ep:star'
      }
    }
  ]
}
```

### Creating New Views
Follow the template pattern:
1. Import `ContentWrap` and `useAxios`
2. Use Element Plus components
3. Add proper TypeScript typing
4. Handle loading states
5. Show success/error messages

## Migration from Simple Panel
The original simple Vue panel (`panel-vue`) has been superseded by this professional template implementation. Key improvements:
- Better UI/UX with professional design
- More consistent code structure
- Better TypeScript support
- Optimized build process
- Production-ready components

## Notes
- Default language set to English (not Chinese)
- All demo components removed for clean IPTV focus
- Icons use Iconify format: `vi-ep:icon-name` or `vi-ant-design:icon-name`
- Session cookies used for authentication (withCredentials: true)
- Dev server auto-reloads on file changes

## Troubleshooting

### 401 Unauthorized
- Check backend is running on port 8080
- Verify credentials: admin / admin123
- Clear browser cookies and re-login

### Module Not Found
- Run `npm install --legacy-peer-deps`
- Clear `node_modules` and reinstall if needed

### Build Errors
- Check TypeScript errors: `npm run type-check`
- Fix ESLint issues: `npm run lint:fix`

## Production Deployment
1. Build the project: `npm run build:pro`
2. Files generated in `dist/` folder
3. Configure Go backend to serve from `dist/`
4. Set up proxy for `/api` and `/stream` routes
