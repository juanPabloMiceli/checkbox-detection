import { Component, computed, effect, input, signal, viewChild } from '@angular/core';
import { Detection } from '../models/detection.model';
import { UploadedDocument } from '../models/uploaded_document.model';
import { ImageViewport } from '../image-viewport/image-viewport';
import { DetectionResults } from '../detection-results/detection-results';

@Component({
  selector: 'app-document-viewer',
  standalone: true,
  imports: [ImageViewport, DetectionResults],
  templateUrl: './document-viewer.html',
  styleUrl: './document-viewer.scss',
})
export class DocumentViewer {
  readonly imageUrl = input.required<string>();
  readonly detections = input.required<Detection[]>();
  readonly status = input.required<UploadedDocument['status']>();
  readonly selectedDetectionIndex = signal<number | null>(null);
  readonly showBoundingBoxes = signal(true);
  private readonly viewport = viewChild(ImageViewport);

  constructor() {
    effect(() => {
      this.imageUrl();
      this.selectedDetectionIndex.set(null);
    });
  }

  totalDetections = computed(() => this.detections().length);
  totalCheckedDetections = computed(
    () => this.detections().filter((detection) => detection.is_checked).length,
  );
  totalUncheckedDetections = computed(() => this.totalDetections() - this.totalCheckedDetections());

  selectDetection(index: number) {
    if (this.selectedDetectionIndex() === index) {
      this.clearSelection();
      return;
    }
    if (this.showBoundingBoxes()) {
      this.viewport()?.inspectSelection();
    }
    this.selectedDetectionIndex.set(index);
  }

  clearSelection() {
    this.selectedDetectionIndex.set(null);
  }

  toggleBoundingBoxes() {
    this.showBoundingBoxes.update((current) => !current);
  }

  nextDetection() {
    const total = this.totalDetections();

    if (total === 0) return;

    const current = this.selectedDetectionIndex();

    const next = current === null ? 0 : (current + 1) % total;

    this.selectedDetectionIndex.set(next);
  }

  previousDetection() {
    const total = this.totalDetections();

    if (total === 0) return;

    const current = this.selectedDetectionIndex();

    const previous = current === null ? total - 1 : (current - 1 + total) % total;

    this.selectedDetectionIndex.set(previous);
  }
}
