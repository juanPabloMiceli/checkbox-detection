import { Component, computed, input, signal } from '@angular/core';
import { Detection } from '../models/detection.model';
import { UploadedDocument } from '../models/uploaded_document.model';
@Component({
  selector: 'app-document-viewer', standalone: true,
  templateUrl: './document-viewer.html', styleUrl: './document-viewer.scss',
})
export class DocumentViewer {
  readonly imageUrl = input.required<string>();
  readonly detections = input.required<Detection[]>();
  readonly status = input.required<UploadedDocument['status']>();
  readonly imageWidth = signal(0);
  readonly imageHeight = signal(0);
  readonly checkedCount = computed(() => this.detections().filter(box => box.is_checked).length);
  onImageLoad(event: Event) {
    const image = event.target as HTMLImageElement;
    this.imageWidth.set(image.naturalWidth);
    this.imageHeight.set(image.naturalHeight);
  }
}
