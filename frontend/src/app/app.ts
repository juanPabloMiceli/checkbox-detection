import { Component, signal, inject, HostListener } from '@angular/core';
import { DocumentViewer } from './document-viewer/document-viewer';
import { DocumentWorkspace } from './services/document-workspace.service';
interface ToastMessage { title: string; description: string; }
@Component({
  selector: 'app-root', standalone: true, imports: [DocumentViewer],
  providers: [DocumentWorkspace], templateUrl: './app.html', styleUrl: './app.scss',
})
export class App {
  readonly workspace = inject(DocumentWorkspace);
  readonly isDragging = signal(false);
  readonly toast = signal<ToastMessage | null>(null);
  onFileSelected(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files?.length) return;
    this.addFiles(Array.from(input.files));
    input.value = '';
  }
  onDrop(event: DragEvent) {
    event.preventDefault(); this.isDragging.set(false);
    this.addFiles(Array.from(event.dataTransfer?.files ?? []));
  }
  addFiles(files: File[]) {
    const rejectedNames = this.workspace.addFiles(files);
    if (rejectedNames.length) this.toast.set({
      title: `${rejectedNames.length} file(s) couldn't be added`,
      description: `Unsupported files: ${rejectedNames.join(', ')}. Only images are supported.`,
    });
  }
  onDragOver(event: DragEvent) { event.preventDefault(); this.isDragging.set(true); }
  onDragLeave(event: DragEvent) {
    event.preventDefault();
    const container = event.currentTarget as HTMLElement;
    if (event.relatedTarget && container.contains(event.relatedTarget as Node)) return;
    this.isDragging.set(false);
  }
  @HostListener('window:dragover', ['$event'])
  preventWindowDragOver(event: DragEvent) { event.preventDefault(); }
  @HostListener('window:drop', ['$event'])
  preventWindowDrop(event: DragEvent) { event.preventDefault(); }
}
