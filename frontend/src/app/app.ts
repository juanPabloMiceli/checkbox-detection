import { Component, signal, inject, HostListener } from '@angular/core';
import { DocumentViewer } from './document-viewer/document-viewer';
import { UploadArea } from './upload-area/upload-area';
import { UploadedDocumentList } from './uploaded-document-list/uploaded-document-list';
import { DocumentWorkspace } from './services/document-workspace.service';

interface ToastMessage {
  title: string;
  description: string;
}

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [DocumentViewer, UploadArea, UploadedDocumentList],
  providers: [DocumentWorkspace],
  templateUrl: './app.html',
  styleUrl: './app.scss',
})
export class App {
  readonly workspace = inject(DocumentWorkspace);

  toast = signal<ToastMessage | null>(null);

  addFiles(files: File[]) {
    const rejectedNames = this.workspace.addFiles(files);
    if (rejectedNames.length > 0) {
      this.toast.set({
        title: `${rejectedNames.length} file(s) couldn't be added`,
        description: `Unsupported files: ${rejectedNames.join(', ')}. Only images are supported.`,
      });
    }
  }

  @HostListener('window:dragover', ['$event'])
  preventWindowDragOver(event: DragEvent) {
    event.preventDefault();
  }

  @HostListener('window:drop', ['$event'])
  preventWindowDrop(event: DragEvent) {
    event.preventDefault();
  }
}
