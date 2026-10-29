import { Component, output, signal, computed, input, effect, ElementRef, afterRenderEffect, viewChild, afterNextRender, DestroyRef, inject } from '@angular/core';
import { Detection } from '../models/detection.model';
import { UploadedDocument } from '../models/uploaded_document.model'
@Component({
  selector: 'app-image-viewport',
  standalone: true,
  templateUrl: './image-viewport.html',
  styleUrl: './image-viewport.scss',
})
export class ImageViewport {

  constructor() {
    effect(() => {
      this.imageUrl();
      this.imageError.set(false);
      this.imageWidth.set(0);
      this.imageHeight.set(0);
    });

    afterRenderEffect({
      write: () => {
        this.zoom();
        this.scrollToSelectedCheckbox();
      },
    });

    afterNextRender(() => {
      const stage = this.imageStage()?.nativeElement;

      if (!stage) return;

      this.resizeObserver = new ResizeObserver(() => {
        this.updateFitScale();
      });

      this.resizeObserver.observe(stage);
      this.updateFitScale();
    });

    this.destroyRef.onDestroy(() => {
      this.resizeObserver?.disconnect();
    });
  }

  zoom = signal(1);

  readonly minZoom = 0.5;
  readonly maxZoom = 4;
  readonly zoomStep = 0.25;
  readonly inspectionZoom = 2;

  imageStage = viewChild<ElementRef<HTMLElement>>('imageStage');
  imageWrapper = viewChild<ElementRef<HTMLElement>>('imageWrapper');

  fitScale = signal(1);

  displayWidth = computed(() =>
    this.imageWidth() * this.fitScale() * this.zoom()
  );

  displayHeight = computed(() =>
    this.imageHeight() * this.fitScale() * this.zoom()
  );

  zoomPercentage = computed(() =>
    Math.round(this.zoom() * 100)
  );

  zoomIn() {
    this.zoom.update(current => Math.min(this.maxZoom, current + this.zoomStep));
  }

  zoomOut() {
    this.zoom.update(current => Math.max(this.minZoom, current - this.zoomStep));
  }

  resetZoom() {
    this.zoom.set(1);
  }

  private resizeObserver?: ResizeObserver;

  private readonly destroyRef = inject(DestroyRef);

  private updateFitScale() {
    const stage = this.imageStage()?.nativeElement;
    const width = this.imageWidth();
    const height = this.imageHeight();

    if (!stage || width === 0 || height === 0) return;

    const availableWidth = stage.clientWidth - 32;
    const availableHeight = stage.clientHeight - 32;

    if (availableWidth <= 0 || availableHeight <= 0) return;

    const scale = Math.min(
      availableWidth / width,
      availableHeight / height
    );

    this.fitScale.set(scale);
  }

  private scrollToSelectedCheckbox() {
    const detection = this.selectedDetection();
    const stage = this.imageStage()?.nativeElement;
    const wrapper = this.imageWrapper()?.nativeElement;

    if (!this.showBoundingBoxes() || !detection || !stage || !wrapper ||
        this.imageWidth() === 0 || this.imageHeight() === 0) return;

    const [x1, y1, x2, y2] = detection.bbox;

    const centerX = (x1 + x2) / 2;
    const centerY = (y1 + y2) / 2;

    const wrapperRect = wrapper.getBoundingClientRect();
    const stageRect = stage.getBoundingClientRect();

    const scaleX = wrapperRect.width / this.imageWidth();
    const scaleY = wrapperRect.height / this.imageHeight();

    const checkboxX =
      wrapperRect.left - stageRect.left +
      centerX * scaleX;

    const checkboxY =
      wrapperRect.top - stageRect.top +
      centerY * scaleY;

    stage.scrollTo({
      left: stage.scrollLeft + checkboxX - stage.clientWidth / 2,
      top: stage.scrollTop + checkboxY - stage.clientHeight / 2,
      behavior: 'instant',
    });
  }

  imageUrl = input.required<string>();
  detections = input.required<Detection[]>();
  status = input.required<UploadedDocument['status']>();
  selectedDetectionIndex = input.required<number | null>();
  showBoundingBoxes = input.required<boolean>();

  readonly detectionSelected = output<number>();
  readonly selectionCleared = output<void>();
  readonly annotationsToggled = output<void>();

  imageError = signal(false);

  imageWidth = signal(0);
  imageHeight = signal(0);

  selectedDetection = computed(() => {
    const index = this.selectedDetectionIndex();
    if (index === null) {
      return null
    };
    return this.detections()[index] ?? null;
  })

  inspectSelection() {
    this.zoom.update(current => Math.max(current, this.inspectionZoom));
  }

  onImageLoad(event: Event) {
    const image = event.target as HTMLImageElement;
    this.imageError.set(false);

    this.imageWidth.set(image.naturalWidth);
    this.imageHeight.set(image.naturalHeight);

    this.resetZoom();
    this.updateFitScale();
  }

  onImageError() {
    this.imageError.set(true);
    this.imageWidth.set(0);
    this.imageHeight.set(0);
  }

  onImageWheel(event: WheelEvent) {
    if (!event.ctrlKey || this.imageError() || this.imageWidth() === 0) return;

    event.preventDefault();

    const stage = this.imageStage()?.nativeElement;
    const wrapper = this.imageWrapper()?.nativeElement;

    if (!stage || !wrapper) return;

    const oldZoom = this.zoom();

    const factor = Math.exp(-event.deltaY * 0.01);

    const newZoom = Math.min(
      this.maxZoom,
      Math.max(this.minZoom, oldZoom * factor)
    );

    if (newZoom === oldZoom) return;

    // Posición del cursor relativa a la imagen actual.
    const rect = wrapper.getBoundingClientRect();

    const relativeX = (event.clientX - rect.left) / rect.width;
    const relativeY = (event.clientY - rect.top) / rect.height;

    // Guardamos el punto que queremos mantener fijo.
    const anchorX = event.clientX;
    const anchorY = event.clientY;

    this.zoom.set(newZoom);

    // Esperamos a que Angular actualice el tamaño del wrapper.
    requestAnimationFrame(() => {
      const newRect = wrapper.getBoundingClientRect();

      const targetX = newRect.left + relativeX * newRect.width;
      const targetY = newRect.top + relativeY * newRect.height;

      stage.scrollLeft += targetX - anchorX;
      stage.scrollTop += targetY - anchorY;
    });
  }

}
