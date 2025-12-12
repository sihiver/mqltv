# Rencana Integrasi IPTV Panel ke Vue Element Plus Admin Template

## Status
🔄 **Dalam Proses** - Template sedang di-install

## Template yang Digunakan
- **Repository**: [vue-element-plus-admin](https://github.com/kailong321200875/vue-element-plus-admin)
- **Versi**: v2.10.0 (Latest)
- **Tech Stack**: Vue 3 + TypeScript + Element Plus + Vite + Pinia
- **Lokasi**: `/home/dindin/mqltv/pannel`

## Fitur Template
✅ Layout profesional dengan sidebar & header
✅ Dynamic routing & permission system
✅ Theming (Dark/Light mode)
✅ Internationalization (i18n)
✅ Multiple layout modes
✅ Breadcrumb & tabs navigation
✅ Full screen mode
✅ Component library
✅ Mock data support
✅ TypeScript support

## Rencana Integrasi

### 1. Setup Awal
- [x] Clone template ke folder `pannel`
- [ ] Install dependencies
- [ ] Test run development server
- [ ] Konfigurasi proxy ke Go backend (port 8080)

### 2. Migrasi Komponen IPTV
File yang akan dimigras dari `panel-vue`:

#### Services & Stores
- [ ] `src/services/api.js` → Sesuaikan dengan struktur template
- [ ] `src/stores/auth.js` → Integrate dengan auth system template
- [ ] `src/stores/stats.js` → Add to stores

#### Views & Components
- [ ] Login (sudah ada di template, customize untuk IPTV)
- [ ] Dashboard → `src/views/iptv/Dashboard.vue`
- [ ] Playlists → `src/views/iptv/Playlists.vue`
- [ ] Channels → `src/views/iptv/Channels.vue`
- [ ] Relays → `src/views/iptv/Relays.vue`
- [ ] Users → `src/views/iptv/Users.vue`
- [ ] Generate Playlist → `src/views/iptv/GeneratePlaylist.vue`
- [ ] Import M3U → `src/views/iptv/ImportM3U.vue`

### 3. Routing Configuration
Tambahkan routes IPTV ke `src/router/routes.ts`:
```typescript
{
  path: '/iptv',
  component: Layout,
  name: 'IPTV',
  meta: {
    title: 'IPTV Management',
    icon: 'ep:video-camera'
  },
  children: [
    {
      path: 'dashboard',
      component: () => import('@/views/iptv/Dashboard.vue'),
      name: 'IPTVDashboard',
      meta: { title: 'Dashboard' }
    },
    {
      path: 'playlists',
      component: () => import('@/views/iptv/Playlists.vue'),
      name: 'Playlists',
      meta: { title: 'Playlists' }
    },
    {
      path: 'channels',
      component: () => import('@/views/iptv/Channels.vue'),
      name: 'Channels',
      meta: { title: 'Channels' }
    },
    // ... dst
  ]
}
```

### 4. API Configuration
Update `vite.config.ts`:
```typescript
server: {
  port: 5173,
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true
    }
  }
}
```

### 5. Authentication Integration
- [ ] Update auth service untuk menggunakan session cookies
- [ ] Customize login page dengan branding IPTV
- [ ] Configure permission system

### 6. Styling & Branding
- [ ] Update theme colors sesuai IPTV branding (purple gradient)
- [ ] Customize sidebar icons
- [ ] Add IPTV logo
- [ ] Update page titles

### 7. Testing
- [ ] Test semua routes
- [ ] Test API integration
- [ ] Test authentication flow
- [ ] Test CRUD operations untuk semua modules
- [ ] Test responsive design

### 8. Build & Deploy
- [ ] Build production
- [ ] Update Go server untuk serve dari dist folder
- [ ] Test production build

## Keuntungan Menggunakan Template

### UI/UX Improvements
✅ **Professional Layout** - Sidebar navigation, header, breadcrumb
✅ **Better Organization** - Structured routing & views
✅ **Responsive Design** - Mobile-friendly out of the box
✅ **Theme Support** - Dark/Light mode
✅ **Better Components** - Rich component library
✅ **Loading States** - Better UX with loading indicators
✅ **Error Handling** - Centralized error handling

### Developer Experience
✅ **TypeScript** - Better type safety
✅ **Code Structure** - Well-organized folders
✅ **Hot Reload** - Fast development
✅ **ESLint/Prettier** - Code quality
✅ **Mock Data** - Easy testing
✅ **Documentation** - Well documented

### Features
✅ **Permission System** - Role-based access control
✅ **Multi-language** - i18n support
✅ **Full Screen Mode** - Better viewing
✅ **Tabs Navigation** - Multiple page tabs
✅ **Breadcrumb** - Easy navigation
✅ **Settings Panel** - Customizable UI

## Next Steps

1. Tunggu `npm install` selesai
2. Test run template dengan `npm run dev`
3. Mulai migrasi komponen satu per satu
4. Test integrasi dengan backend
5. Polish & optimize

## Catatan
- Template ini lebih kompleks dari panel sederhana sebelumnya
- Lebih cocok untuk produksi & scalability
- Learning curve lebih tinggi tapi worth it
- Bisa digunakan untuk project lain di masa depan
