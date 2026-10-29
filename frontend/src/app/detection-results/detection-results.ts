import { afterRenderEffect, Component, computed, ElementRef, input, output, viewChildren } from '@angular/core';
import { Detection } from '../models/detection.model';

@Component({
  selector: 'app-detection-results',
  standalone: true,
  templateUrl: './detection-results.html',
  styleUrl: './detection-results.scss',
})
export class DetectionResults {
  readonly detections = input.required<Detection[]>();
  readonly selectedDetectionIndex = input.required<number | null>();
  readonly showBoundingBoxes = input.required<boolean>();
  readonly zoom = input.required<number>();
  readonly totalDetections = computed(() => this.detections().length);

  readonly detectionSelected = output<number>();
  readonly previousRequested = output<void>();
  readonly nextRequested = output<void>();

  private readonly detectionButtons = viewChildren<ElementRef<HTMLButtonElement>>('detectionButton');

  constructor() {
    afterRenderEffect({
      write: () => {
        this.zoom();
        const index = this.selectedDetectionIndex();
        if (index === null || !this.showBoundingBoxes()) return;
        this.detectionButtons()[index]?.nativeElement.scrollIntoView({
          behavior: 'smooth',
          block: 'nearest',
        });
      },
    });
  }
}
