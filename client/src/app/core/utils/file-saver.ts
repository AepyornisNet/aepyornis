import { HttpResponse } from '@angular/common/http';

/**
 * Triggers a browser download of a Blob as a file with the given filename.
 *
 * @param blob The Blob data to download
 * @param filename The suggested filename for the downloaded file
 */
export function saveBlob(blob: Blob, filename: string): void {
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.URL.revokeObjectURL(url);
}

/**
 * Extracts a filename from the HTTP Content-Disposition header if available.
 *
 * @param contentDisposition The Content-Disposition header value
 * @param fallback The fallback filename if no header or filename is found
 * @returns The extracted filename or fallback
 */
export function getFilenameFromContentDisposition(
  contentDisposition?: string | null,
  fallback = 'download',
): string {
  if (!contentDisposition) {
    return fallback;
  }

  // Handle filename* (RFC 5987 / RFC 6266 utf-8 encoded filename)
  const filenameStarMatch = contentDisposition.match(/filename\*=utf-8''([^;]+)/i);
  if (filenameStarMatch && filenameStarMatch[1]) {
    try {
      return decodeURIComponent(filenameStarMatch[1]);
    } catch {
      // Fall back to standard filename match if decoding fails
    }
  }

  // Handle standard filename="name" or filename=name
  const filenameMatch = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/i);
  if (filenameMatch && filenameMatch[1]) {
    return filenameMatch[1].replace(/['"]/g, '').trim();
  }

  return fallback;
}

/**
 * Saves a file from an HttpResponse containing a Blob body, extracting the filename
 * from Content-Disposition header if present, or falling back to the specified fallback.
 *
 * @param response The HTTP response containing the Blob body
 * @param fallbackFilename The fallback filename if none is specified in response headers
 */
export function saveHttpResponse(
  response: HttpResponse<Blob>,
  fallbackFilename = 'download',
): void {
  if (!response.body) {
    return;
  }
  const filename = getFilenameFromContentDisposition(
    response.headers.get('content-disposition'),
    fallbackFilename,
  );
  saveBlob(response.body, filename);
}
