#!/bin/bash

# Script untuk build production Vue app

cd /home/dindin/mqltv/panel-vue

echo "🏗️  Building Vue.js for Production..."
echo ""

npm run build

echo ""
echo "✅ Build complete! Files are in /home/dindin/mqltv/dist"
echo ""
echo "Update your Go server to serve from the dist folder:"
echo "  http.Handle(\"/\", http.FileServer(http.Dir(\"./dist\")))"
