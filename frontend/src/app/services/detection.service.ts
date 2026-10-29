import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { DetectionResponse } from '../models/detection.model';
import { Observable } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class DetectionService {
  private http = inject(HttpClient);
  private readonly apiUrl = 'http://localhost:8080/api/detect';

  detectCheckboxes(file: File): Observable<DetectionResponse> {
    const formData = new FormData();
    formData.append('image', file);

    return this.http.post<DetectionResponse>(this.apiUrl, formData);
  }
}
