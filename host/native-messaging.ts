// Chrome native messaging framing: each message is a 4-byte little-endian
// length followed by that many bytes of UTF-8 JSON, on stdin/stdout.

const MAX_INCOMING = 1024 * 1024; // the extension only sends small song updates

export function encodeNativeMessage(message: unknown): Buffer {
  const body = Buffer.from(JSON.stringify(message), 'utf8');
  const header = Buffer.alloc(4);
  header.writeUInt32LE(body.length, 0);
  return Buffer.concat([header, body]);
}

export function readNativeMessages(
  input: NodeJS.ReadableStream,
  onMessage: (message: unknown) => void,
  onError: (error: Error) => void,
): void {
  let buffer: Buffer = Buffer.alloc(0);
  input.on('data', (chunk: Buffer) => {
    buffer = Buffer.concat([buffer, chunk]);
    while (buffer.length >= 4) {
      const length = buffer.readUInt32LE(0);
      if (length > MAX_INCOMING) {
        onError(new Error(`message too large: ${length} bytes`));
        buffer = Buffer.alloc(0);
        return;
      }
      if (buffer.length < 4 + length) return;
      const body = buffer.subarray(4, 4 + length).toString('utf8');
      buffer = buffer.subarray(4 + length);
      let message: unknown;
      try {
        message = JSON.parse(body);
      } catch (error) {
        onError(error as Error);
        continue;
      }
      onMessage(message);
    }
  });
}
