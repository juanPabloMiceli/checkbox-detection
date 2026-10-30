import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { DetectionResponse } from '../models/detection.model';
import { DetectionSettings, SettingsDefinition } from '../models/detection-settings.model';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class DetectionService {
  private http = inject(HttpClient);
  private readonly apiUrl = '/detect';

  getSettings(): Observable<SettingsDefinition> {
    return this.http.get<SettingsDefinition>(`${this.apiUrl}/settings`);
  }

  detectCheckboxes(file: File, settings?: DetectionSettings): Observable<DetectionResponse> {
    const formData = new FormData();
    formData.append('image', file);
    if (settings) formData.append('settings', JSON.stringify(settings));

    return this.http.post<DetectionResponse>(this.apiUrl, formData);
  }
}
