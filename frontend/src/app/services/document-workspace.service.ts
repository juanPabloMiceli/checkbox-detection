import { computed, inject, Injectable, OnDestroy, signal } from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';
import { finalize, Subscription } from 'rxjs';
import { UploadedDocument } from '../models/uploaded_document.model';
import { DetectionSettingsStore } from './detection-settings.service';
import { DetectionService } from './detection.service';

// Provided by App so requests and object URLs live only as long as the workspace.
@Injectable()
export class DocumentWorkspace implements OnDestroy {
  private readonly detectionService = inject(DetectionService);
  private readonly activeRequests = new Map<number, Subscription>();
  readonly settings = inject(DetectionSettingsStore);
  private nextDocumentId = 1;

  private readonly documentState = signal<UploadedDocument[]>([]);
  private readonly selectionState = signal<number | null>(null);

  readonly documents = this.documentState.asReadonly();
  readonly selectedDocumentId = this.selectionState.asReadonly();
  readonly selectedDocument = computed(
    () => this.documents().find((document) => document.id === this.selectedDocumentId()) ?? null,
  );

  addFiles(files: File[]): string[] {
    const newDocuments: UploadedDocument[] = files
      .filter((file) => file.type.startsWith('image/'))
      .map((file) => ({
        id: this.nextDocumentId++,
        name: file.name,
        file,
        imageUrl: URL.createObjectURL(file),
        status: 'pending',
        detections: [],
      }));

    this.documentState.update((current) => [...current, ...newDocuments]);
    if (newDocuments.length > 0) {
      this.selectionState.set(newDocuments[0].id);
    }

    return files.filter((file) => !file.type.startsWith('image/')).map((file) => file.name);
  }

  selectDocument(documentId: number) {
    this.selectionState.set(documentId);
  }

  processDocument(documentId: number) {
    const document = this.documents().find((doc) => doc.id === documentId);
    const settings = this.settings.snapshot();
    if (!document || document.status === 'processing' || !settings) return;

    this.updateDocument(documentId, {
      detections: [],
      status: 'processing',
      errorMessage: undefined,
    });

    const subscription = this.detectionService
      .detectCheckboxes(document.file, settings)
      .pipe(finalize(() => this.activeRequests.delete(documentId)))
      .subscribe({
        next: (response) => {
          this.updateDocument(documentId, { detections: response.boxes, status: 'completed' });
        },
        error: (error: HttpErrorResponse) => {
          this.updateDocument(documentId, {
            detections: [],
            status: 'error',
            errorMessage: this.getErrorMessage(error),
          });
        },
      });

    // A synchronous response can complete before subscribe returns.
    if (!subscription.closed) {
      this.activeRequests.set(documentId, subscription);
    }
  }

  processAll() {
    this.documents().forEach((document) => this.processDocument(document.id));
  }

  removeDocument(documentId: number) {
    const document = this.documents().find((document) => document.id === documentId);
    if (!document) return;

    this.documentState.update((documents) =>
      documents.filter((document) => document.id !== documentId),
    );
    this.selectionState.update((selectedId) => (selectedId === documentId ? null : selectedId));
    URL.revokeObjectURL(document.imageUrl);
    this.activeRequests.get(documentId)?.unsubscribe();
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

  private updateDocument(documentId: number, changes: Partial<UploadedDocument>) {
    this.documentState.update((documents) =>
      documents.map((document) =>
        document.id === documentId ? { ...document, ...changes } : document,
      ),
    );
  }

  private getErrorMessage(error: HttpErrorResponse): string {
    switch (error.status) {
      case 0:
        return 'Cannot connect to the detection server.';
      case 400:
        return 'The uploaded image is invalid.';
      case 413:
        return 'The image is too large.';
      case 503:
        return 'The detection backend is not available yet. Please try again later.';
      case 500:
        return 'The detection server encountered an error.';
      default:
        return `Processing failed (HTTP ${error.status}).`;
    }
  }
}
