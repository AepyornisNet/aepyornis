import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideZonelessChangeDetection } from '@angular/core';
import { provideIcons } from '@ng-icons/core';
import { faSolidFile, faSolidXmark } from '@ng-icons/font-awesome/solid';
import { TranslateModule } from '@ngx-translate/core';
import { FileUploadList } from './file-upload-list';

describe('FileUploadList', () => {
  let component: FileUploadList;
  let fixture: ComponentFixture<FileUploadList>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FileUploadList, TranslateModule.forRoot()],
      providers: [provideZonelessChangeDetection(), provideIcons({ faSolidFile, faSolidXmark })],
    }).compileComponents();

    fixture = TestBed.createComponent(FileUploadList);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize with default empty items', () => {
    expect(component.items()).toEqual([]);
  });

  it('should format file sizes correctly', () => {
    expect(component.formatFileSize(500)).toBe('500 B');
    expect(component.formatFileSize(1024 * 5)).toBe('5.0 KB');
    expect(component.formatFileSize(1024 * 1024 * 2.5)).toBe('2.5 MB');
  });

  it('should prefill name from filename when optionalName is false', () => {
    fixture.componentRef.setInput('optionalName', false);
    fixture.detectChanges();

    const file = new File(['dummy content'], 'morning_ride.gpx', { type: 'application/gpx+xml' });
    const event = {
      target: {
        files: [file],
        value: '',
      },
    } as unknown as Event;

    component.onFilesSelected(event);

    expect(component.items().length).toBe(1);
    expect(component.items()[0].file.name).toBe('morning_ride.gpx');
    expect(component.items()[0].name).toBe('morning_ride');
  });

  it('should keep name empty for auto-generation when optionalName is true', () => {
    fixture.componentRef.setInput('optionalName', true);
    fixture.detectChanges();

    const file = new File(['dummy content'], 'activity.fit', { type: 'application/octet-stream' });
    const event = {
      target: {
        files: [file],
        value: '',
      },
    } as unknown as Event;

    component.onFilesSelected(event);

    expect(component.items().length).toBe(1);
    expect(component.items()[0].file.name).toBe('activity.fit');
    expect(component.items()[0].name).toBe('');
  });

  it('should allow updating item name', () => {
    const file = new File(['dummy content'], 'test.gpx', { type: 'application/gpx+xml' });
    component.items.set([{ file, name: 'old_name' }]);

    component.updateItemName(0, 'new_custom_name');

    expect(component.items()[0].name).toBe('new_custom_name');
  });

  it('should allow removing item', () => {
    const file1 = new File(['1'], 'file1.gpx', { type: 'application/gpx+xml' });
    const file2 = new File(['2'], 'file2.gpx', { type: 'application/gpx+xml' });
    component.items.set([
      { file: file1, name: 'f1' },
      { file: file2, name: 'f2' },
    ]);

    component.removeItem(0);

    expect(component.items().length).toBe(1);
    expect(component.items()[0].name).toBe('f2');
  });
});
