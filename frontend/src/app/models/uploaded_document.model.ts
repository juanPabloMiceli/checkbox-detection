import { Detection } from './detection.model';

export interface UploadedDocument {
  id: number;
  name: string;
  file: File;
  imageUrl: string;
  status: 'pending' | 'processing' | 'completed' | 'error';
  detections: Detection[];
  errorMessage?: string;
}
