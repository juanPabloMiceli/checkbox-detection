import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { DetectionResponse } from '../models/detection.model';
import { DetectionSettings, SettingsDefinition } from '../models/detection-settings.model';
import { Observable, retry, throwError, timer } from 'rxjs';

// Four retries after the initial request; equal jitter avoids synchronized uploads.
function retryWhenOverloaded<T>() {
  return retry<T>({
    count: 4,
    delay: (error: unknown, retryNumber) => {
      if (!(error instanceof HttpErrorResponse) || ![429, 502, 503].includes(error.status)) {
        return throwError(() => error);
      }

      const backoff = Math.min(1000 * 2 ** (retryNumber - 1), 8000);
      const retryAfter = error.headers.get('Retry-After');
      let serverDelay = 0;
      if (retryAfter) {
        const seconds = Number(retryAfter);
        serverDelay = Number.isFinite(seconds)
          ? Math.max(0, seconds * 1000)
          : Math.max(0, Date.parse(retryAfter) - Date.now());
      }

      // Fail normally rather than leave the document waiting indefinitely.
      if (serverDelay > 60000) return throwError(() => error);
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
    return this.http.get<SettingsDefinition>(`${this.apiUrl}/settings`).pipe(retryWhenOverloaded());
  }

  detectCheckboxes(file: File, settings?: DetectionSettings): Observable<DetectionResponse> {
    const formData = new FormData();
    formData.append('image', file);
    if (settings) formData.append('settings', JSON.stringify(settings));

    return this.http.post<DetectionResponse>(this.apiUrl, formData).pipe(retryWhenOverloaded());
  }
}
