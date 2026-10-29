import { Component, output, signal } from '@angular/core';

@Component({
  selector: 'app-upload-area',
  standalone: true,
  templateUrl: './upload-area.html',
  styleUrl: './upload-area.scss',
})
export class UploadArea {
  readonly filesAdded = output<File[]>();
  readonly isDragging = signal(false);

  onFileSelected(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files?.length) return;

    this.filesAdded.emit(Array.from(input.files));
    input.value = '';
  }

  onDrop(event: DragEvent) {
    event.preventDefault();
    this.isDragging.set(false);
    this.filesAdded.emit(Array.from(event.dataTransfer?.files ?? []));
  }

  onDragOver(event: DragEvent) {
    event.preventDefault();
    this.isDragging.set(true);
  }

  onDragLeave(event: DragEvent) {
    event.preventDefault();
    const container = event.currentTarget as HTMLElement;
    const nextElement = event.relatedTarget as Node | null;
    if (nextElement && container.contains(nextElement)) return;

    this.isDragging.set(false);
  }
}
