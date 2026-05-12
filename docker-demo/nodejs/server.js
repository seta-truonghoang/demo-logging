const http = require('http');
const os = require('os');

const PORT = process.env.PORT || 8081;
const HOSTNAME = os.hostname();

const server = http.createServer((req, res) => {
  // Log the incoming request
  console.log(`[${new Date().toISOString()}] ${req.method} ${req.url}`);

  if (req.url === '/' || req.url === '/api/info') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    const response = {
      language: 'Node.js',
      message: 'Xin chào từ Docker Container tối ưu cho Node.js!',
      timestamp: new Date().toISOString(),
      hostname: HOSTNAME,
      version: '1.0.0'
    };
    res.end(JSON.stringify(response));
  } else {
    res.writeHead(404, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: 'Not Found' }));
  }
});

// Handle shutdown signals from Docker (SIGTERM and SIGINT) for a Graceful Shutdown
const gracefulShutdown = () => {
  console.log('Shutting down gracefully...');
  server.close(() => {
    console.log('Closed out remaining connections');
    process.exit(0);
  });

  // Force shutdown after 10 seconds if connections are still hanging
  setTimeout(() => {
    console.error('Could not close connections in time, forcefully shutting down');
    process.exit(1);
  }, 10000);
};

process.on('SIGTERM', gracefulShutdown);
process.on('SIGINT', gracefulShutdown);

server.listen(PORT, () => {
  console.log(`Node.js application is starting on port ${PORT}...`);
});
