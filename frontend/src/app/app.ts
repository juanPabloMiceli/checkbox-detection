import { OnDestroy, Component, signal, inject, computed, HostListener } from '@angular/core';
import { DocumentViewer } from './document-viewer/document-viewer';
import { UploadedDocument } from './models/uploaded_document.model';
import { Subscription, finalize } from 'rxjs';
import { DetectionService } from './services/detection.service';
import { HttpErrorResponse } from '@angular/common/http';

interface ToastMessage {
  title: string;
  description: string;
}

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [DocumentViewer],
  templateUrl: './app.html',
  styleUrl: './app.scss',
})
export class App implements OnDestroy {
  private activeRequests = new Map<number, Subscription>();
  private detectionService = inject(DetectionService);
  private nextDocumentId = 1;

  documents = signal<UploadedDocument[]>([]);
  selectedDocumentId = signal<number | null>(null);

  isDragging = signal(false);
  toast = signal<ToastMessage | null>(null);

  onFileSelected(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files?.length) return;

    const files = Array.from(input.files);
    this.addFiles(files);
    input.value = '';
  }

  onDrop(event: DragEvent) {
    event.preventDefault();
    this.isDragging.set(false);
    const files = Array.from(event.dataTransfer?.files ?? []);
    this.addFiles(files);
  }

  private addFiles(files: File[]) {
    const newDocuments: UploadedDocument[] = files
      .filter(file => file.type.startsWith('image/'))
      .map(file => ({
        id: this.nextDocumentId++,
        name: file.name,
        file,
        imageUrl: URL.createObjectURL(file),
        status: 'pending' as const,
        detections: [],
      }));

    const rejectedFiles = files.filter(file => !file.type.startsWith('image/'));
    if (rejectedFiles.length > 0) {
      this.toast.set({
        title: `${rejectedFiles.length} file(s) couldn't be added`,
        description: `Unsupported files: ${rejectedFiles.map(file => file.name).join(', ')}. Only images are supported.`,
      });
    }
    this.documents.update(current => [...current, ...newDocuments]);

    if (newDocuments.length > 0) {
      this.selectedDocumentId.set(newDocuments[0].id);
    }
  }

  detectCheckboxes(documentId: number) {
    const document = this.documents().find(doc => doc.id === documentId);

    if (!document || document.status === 'processing') {
      return;
    }

    this.updateDocument(document.id, {detections: [], status: 'processing', errorMessage: undefined})
    const subscription = this.detectionService.detectCheckboxes(document.file).pipe(
        finalize(() => {
          this.activeRequests.delete(documentId);
        })
      ).subscribe({
        next: response => {
          this.updateDocument(document.id, {detections: response.boxes, status: 'completed'})
        },
        error: (error: HttpErrorResponse) => {
          this.updateDocument(documentId, {
            detections: [],
            status: 'error',
            errorMessage: this.getErrorMessage(error),
          });
        },
      });

    if (!subscription.closed) {
      this.activeRequests.set(documentId, subscription);
    }
  }

  removeDocument(documentId: number) {
    const selectedDocument = this.documents().find(document => document.id === documentId);
    if (!selectedDocument) {
      return;
    }
    this.documents.update(documents => documents.filter(document => document.id !== documentId));
    this.selectedDocumentId.update(selectedId => selectedId === documentId ? null : selectedId);
    URL.revokeObjectURL(selectedDocument.imageUrl);
    const subscription = this.activeRequests.get(documentId);
    if (subscription){
      subscription.unsubscribe();
    }
  }

  private updateDocument(
    documentId: number,
    changes: Partial<UploadedDocument>
  ) {
    this.documents.update(documents =>
      documents.map(document =>
        document.id === documentId
          ? { ...document, ...changes }
          : document
      )
    );
  }

  processAllImages() {
    this.documents().forEach(document => this.detectCheckboxes(document.id));
  }

  ngOnDestroy() {
    for (const subscription of this.activeRequests.values()) {
      subscription.unsubscribe();
    }

    this.activeRequests.clear();

    for (const document of this.documents()) {
      URL.revokeObjectURL(document.imageUrl);
    }
  }

  private getErrorMessage(error: HttpErrorResponse): string {
    switch (error.status) {
      case 0:
        return 'Cannot connect to the detection server.';
      case 400:
        return 'The uploaded image is invalid.';
      case 413:
        return 'The image is too large.';
      case 500:
        return 'The detection server encountered an error.';
      default:
        return `Processing failed (HTTP ${error.status}).`;
    }
  }

  selectNewDocument(documentId: number) {
    this.selectedDocumentId.set(documentId);
  }

  onDragOver(event: DragEvent) {
    event.preventDefault();
    this.isDragging.set(true);
  }

  onDragLeave(event: DragEvent) {
    event.preventDefault();
    const container = event.currentTarget as HTMLElement;
    const nextElement = event.relatedTarget as Node | null;

    if (nextElement && container.contains(nextElement)) {
      return;
    }

    this.isDragging.set(false);
  }

  @HostListener('window:dragover', ['$event'])
  preventWindowDragOver(event: DragEvent) {
    event.preventDefault();
  }

  @HostListener('window:drop', ['$event'])
  preventWindowDrop(event: DragEvent) {
    event.preventDefault();
  }

  selectedDocument = computed(() => this.documents().find(document => document.id === this.selectedDocumentId()) ?? null)
}

