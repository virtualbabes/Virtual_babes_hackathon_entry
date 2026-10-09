// raw_ws_probe.cjs — raw TCP WebSocket handshake, dump bytes on the wire.
const net = require('net');
const key = Buffer.from('0123456789abcdef').toString('base64');
const req =
  'GET /ws HTTP/1.1\r\n' +
  'Host: localhost:8090\r\n' +
  'Upgrade: websocket\r\n' +
  'Connection: Upgrade\r\n' +
  'Sec-WebSocket-Key: ' + key + '\r\n' +
  'Sec-WebSocket-Version: 13\r\n' +
  'Origin: http://localhost:8090\r\n' +
  '\r\n';
const sock = net.connect(8090, '127.0.0.1', () => { sock.write(req); });
let got = 0;
sock.on('data', (d) => {
  got += d.length;
  console.log('--- RAW BYTES (' + d.length + ') hex ---');
  console.log(d.toString('hex').slice(0, 300));
  console.log('--- ASCII ---');
  console.log(d.toString('ascii').slice(0, 300));
});
sock.on('close', () => console.log('CLOSED total=' + got));
sock.on('error', (e) => console.log('ERR ' + e.message));
setTimeout(() => { sock.end(); process.exit(0); }, 5000);
