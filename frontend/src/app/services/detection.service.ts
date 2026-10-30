import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { DetectionResponse } from '../models/detection.model';
import { DetectionSettings, SettingsDefinition } from '../models/detection-settings.model';
import { Observable, retry, throwError, timer } from 'rxjs';

const MAX_RETRIES = 4;
const BASE_RETRY_DELAY_MS = 1000;
const MAX_BACKOFF_MS = 8000;
const MAX_SERVER_WAIT_MS = 60000;

// Add jitter even when Retry-After is present to avoid synchronized uploads.
function retryTransientErrors<T>() {
  return retry<T>({
    count: MAX_RETRIES,
    delay: (error: unknown, retryNumber) => {
      if (!(error instanceof HttpErrorResponse) || ![429, 502, 503].includes(error.status)) {
        return throwError(() => error);
      }

      const backoff = Math.min(BASE_RETRY_DELAY_MS * 2 ** (retryNumber - 1), MAX_BACKOFF_MS);
      const retryAfter = error.headers.get('Retry-After');
      let serverDelay = 0;
      if (retryAfter) {
        const seconds = Number(retryAfter);
        serverDelay = Number.isFinite(seconds)
          ? Math.max(0, seconds * 1000)
          : Math.max(0, Date.parse(retryAfter) - Date.now());
      }

      // Fail normally rather than leave the document waiting indefinitely.
      if (serverDelay > MAX_SERVER_WAIT_MS) return throwError(() => error);
      const delay = Math.max(serverDelay || 0, backoff / 2) + Math.random() * (backoff / 2);
      return timer(delay);
    },
  });
}

@Injectable({
  providedIn: 'root',
})
export class DetectionService {
  private http = inject(HttpClient);
  private readonly apiUrl = '/detect';

  getSettings(): Observable<SettingsDefinition> {
    return this.http
      .get<SettingsDefinition>(`${this.apiUrl}/settings`)
      .pipe(retryTransientErrors());
  }

  detectCheckboxes(file: File, settings?: DetectionSettings): Observable<DetectionResponse> {
    const formData = new FormData();
    formData.append('image', file);
    if (settings) formData.append('settings', JSON.stringify(settings));

    return this.http.post<DetectionResponse>(this.apiUrl, formData).pipe(retryTransientErrors());
  }
}
