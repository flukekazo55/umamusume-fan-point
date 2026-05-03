module.exports = async function proxy(request, response) {
  const backendUrl = process.env.BACKEND_URL;
  if (!backendUrl) {
    response.status(500).json({
      error: 'BACKEND_URL is not configured in Vercel environment variables'
    });
    return;
  }

  const path = Array.isArray(request.query.path)
    ? request.query.path.join('/')
    : request.query.path || '';

  const incomingUrl = new URL(request.url, `https://${request.headers.host || 'localhost'}`);
  const targetUrl = new URL(`/api/${path}`, backendUrl);
  for (const [key, value] of incomingUrl.searchParams.entries()) {
    if (key !== 'path') {
      targetUrl.searchParams.append(key, value);
    }
  }

  const headers = { ...request.headers };
  delete headers.host;
  delete headers.connection;
  delete headers['content-length'];

  const body = ['GET', 'HEAD'].includes(request.method) ? undefined : await readBody(request);
  const backendResponse = await fetch(targetUrl, {
    method: request.method,
    headers,
    body
  });

  response.status(backendResponse.status);
  backendResponse.headers.forEach((value, key) => {
    if (!['content-encoding', 'content-length', 'transfer-encoding'].includes(key.toLowerCase())) {
      response.setHeader(key, value);
    }
  });

  const buffer = Buffer.from(await backendResponse.arrayBuffer());
  response.send(buffer);
};

function readBody(request) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    request.on('data', (chunk) => chunks.push(chunk));
    request.on('end', () => resolve(Buffer.concat(chunks)));
    request.on('error', reject);
  });
}
