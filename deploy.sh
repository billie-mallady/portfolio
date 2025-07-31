#!/bin/bash

# Uberspace deployment script for Portfolio Terminal
# Run this script on your Uberspace server after git pull

set -e

echo "🚀 Starting Portfolio Terminal deployment on Uberspace..."

# Set up environment
export NODE_ENV=production
export PORT=8080

# Install/update Go if needed (Uberspace supports Go 1.19+)
echo "📦 Checking Go installation..."
go version || (echo "❌ Go not found. Please install Go on Uberspace first." && exit 1)

# Build the Go backend
echo "🔨 Building Go backend..."
go build -o portfolio main.go

# Install Node.js dependencies for frontend
echo "📦 Installing frontend dependencies..."
cd frontend
npm install --production

# Build React frontend
echo "🔨 Building React frontend..."
npm run build

# Go back to root
cd ..

echo "✅ Build complete!"

# Set up supervisord for process management
echo "⚙️  Setting up process management..."

# Create supervisord config if it doesn't exist
if [ ! -f ~/etc/services.d/portfolio.ini ]; then
    mkdir -p ~/etc/services.d/
    cat > ~/etc/services.d/portfolio.ini << EOF
[program:portfolio]
command=$(pwd)/portfolio
autostart=yes
autorestart=yes
startsecs=30
startretries=3
redirect_stderr=true
stdout_logfile=$(pwd)/portfolio.log
environment=PORT=8080,NODE_ENV=production,MONGODB_URI="\$MONGODB_URI",OPENAI_API_KEY="\$OPENAI_API_KEY"
EOF
fi

echo "🔄 Restarting services..."
supervisorctl reread
supervisorctl update
supervisorctl restart portfolio

echo "🌐 Setting up web service..."

# Create web service configuration
uberspace web backend set / --http --port 8080

echo "✅ Deployment complete!"
echo "📋 Your application should be available at: https://$(hostname -f)"
echo "📝 Logs available at: $(pwd)/portfolio.log"
echo "🔧 Manage service with: supervisorctl [start|stop|restart] portfolio"
