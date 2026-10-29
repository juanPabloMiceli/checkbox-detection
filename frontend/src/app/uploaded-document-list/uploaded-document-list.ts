import { Component, input, output } from '@angular/core';
import { UploadedDocument } from '../models/uploaded_document.model';

@Component({
  selector: 'app-uploaded-document-list',
  standalone: true,
  templateUrl: './uploaded-document-list.html',
  styleUrl: './uploaded-document-list.scss',
})
export class UploadedDocumentList {
  readonly documents = input.required<UploadedDocument[]>();
  readonly selectedDocumentId = input.required<number | null>();

  readonly documentSelected = output<number>();
  readonly documentProcessed = output<number>();
  readonly documentRemoved = output<number>();
  readonly processAll = output<void>();
}
