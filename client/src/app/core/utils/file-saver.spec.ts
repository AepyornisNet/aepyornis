import { HttpHeaders, HttpResponse } from '@angular/common/http';
import {
  getFilenameFromContentDisposition,
  saveBlob,
  saveHttpResponse,
} from './file-saver';

describe('file-saver', () => {
  describe('getFilenameFromContentDisposition', () => {
    it('should return fallback when header is null or undefined', () => {
      expect(getFilenameFromContentDisposition(null, 'default.txt')).toBe('default.txt');
      expect(getFilenameFromContentDisposition(undefined, 'default.txt')).toBe('default.txt');
      expect(getFilenameFromContentDisposition('', 'default.txt')).toBe('default.txt');
    });

    it('should extract filename from standard header', () => {
      expect(
        getFilenameFromContentDisposition('attachment; filename="workout_123.gpx"'),
      ).toBe('workout_123.gpx');
      expect(
        getFilenameFromContentDisposition('attachment; filename=workout_123.gpx'),
      ).toBe('workout_123.gpx');
    });

    it('should extract filename from RFC 5987 filename* header', () => {
      expect(
        getFilenameFromContentDisposition(
          "attachment; filename*=UTF-8''my%20route%20segment.gpx",
        ),
      ).toBe('my route segment.gpx');
    });
  });

  describe('saveBlob and saveHttpResponse', () => {
    let originalCreateObjectURL: typeof window.URL.createObjectURL;
    let originalRevokeObjectURL: typeof window.URL.revokeObjectURL;
    let createObjectURLSpy: jasmine.Spy;
    let revokeObjectURLSpy: jasmine.Spy;

    beforeEach(() => {
      originalCreateObjectURL = window.URL.createObjectURL;
      originalRevokeObjectURL = window.URL.revokeObjectURL;

      createObjectURLSpy = jasmine
        .createSpy('createObjectURL')
        .and.returnValue('blob:http://localhost/test-uuid');
      revokeObjectURLSpy = jasmine.createSpy('revokeObjectURL');

      window.URL.createObjectURL = createObjectURLSpy;
      window.URL.revokeObjectURL = revokeObjectURLSpy;
    });

    afterEach(() => {
      window.URL.createObjectURL = originalCreateObjectURL;
      window.URL.revokeObjectURL = originalRevokeObjectURL;
    });

    it('should trigger download link creation and cleanup for saveBlob', () => {
      const appendChildSpy = spyOn(document.body, 'appendChild').and.callThrough();
      const removeChildSpy = spyOn(document.body, 'removeChild').and.callThrough();

      const blob = new Blob(['test content'], { type: 'text/plain' });
      saveBlob(blob, 'test.gpx');

      expect(createObjectURLSpy).toHaveBeenCalledWith(blob);
      expect(appendChildSpy).toHaveBeenCalled();
      expect(removeChildSpy).toHaveBeenCalled();
      expect(revokeObjectURLSpy).toHaveBeenCalledWith('blob:http://localhost/test-uuid');
    });

    it('should save HttpResponse with Blob and correct filename', () => {
      const appendChildSpy = spyOn(document.body, 'appendChild').and.callThrough();
      const removeChildSpy = spyOn(document.body, 'removeChild').and.callThrough();

      const blob = new Blob(['workout data'], { type: 'application/gpx+xml' });
      const response = new HttpResponse<Blob>({
        body: blob,
        headers: new HttpHeaders({
          'Content-Disposition': 'attachment; filename="my_workout.gpx"',
        }),
      });

      saveHttpResponse(response, 'fallback.gpx');

      expect(createObjectURLSpy).toHaveBeenCalledWith(blob);
      expect(appendChildSpy).toHaveBeenCalled();
      expect(removeChildSpy).toHaveBeenCalled();
    });

    it('should do nothing if HttpResponse body is null', () => {
      const response = new HttpResponse<Blob>({
        body: null,
      });

      saveHttpResponse(response, 'fallback.gpx');
      expect(createObjectURLSpy).not.toHaveBeenCalled();
    });
  });
});
