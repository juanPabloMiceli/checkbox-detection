import { ComponentFixture, TestBed } from '@angular/core/testing';
import { DocumentViewer } from './document-viewer';

describe('DocumentViewer', () => {
  let component: DocumentViewer;
  let fixture: ComponentFixture<DocumentViewer>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DocumentViewer],
    }).compileComponents();

    fixture = TestBed.createComponent(DocumentViewer);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('imageUrl', '/samples/document.png');
    fixture.componentRef.setInput('detections', []);
    fixture.componentRef.setInput('status', 'pending');
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
