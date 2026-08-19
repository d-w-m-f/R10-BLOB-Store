import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface BlobFile {
  blob_uuid: string;
  blob_filename: string;
  blob_size: number;
  blob_mime_type: string;
  blob_created_at: string;
  blob_deleted?: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class FileService {
  private apiUrl = `${environment.apiGatewayUrl}/api/v1/files`;

  constructor(private http: HttpClient) {}

  listFiles(): Observable<BlobFile[]> {
    return this.http.get<BlobFile[]>(this.apiUrl);
  }

  deleteFile(blobUuid: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${blobUuid}`);
  }

  /**
   * Pulls a blob back out of the cluster. The gateway reassembles it from its
   * chunks (reconstructing missing Reed-Solomon shards when needed) and streams
   * the original bytes, so the browser only ever sees a plain binary response.
   */
  downloadFile(blobUuid: string): Observable<Blob> {
    return this.http.get(`${this.apiUrl}/${blobUuid}/download`, { responseType: 'blob' });
  }

  formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  getFileIcon(filename: string, mimeType?: string): string {
    const lowerName = filename ? filename.toLowerCase() : '';
    if (lowerName.endsWith('.pdf') || mimeType?.includes('pdf')) return 'pdf';
    if (lowerName.endsWith('.zip') || lowerName.endsWith('.tar') || lowerName.endsWith('.gz') || mimeType?.includes('zip') || mimeType?.includes('archive')) return 'zip';
    if (lowerName.match(/\.(jpg|jpeg|png|gif|webp|svg)$/) || mimeType?.includes('image')) return 'image';
    if (lowerName.match(/\.(xls|xlsx|csv)$/) || mimeType?.includes('spreadsheet') || mimeType?.includes('excel')) return 'xls';
    if (lowerName.match(/\.(doc|docx)$/) || mimeType?.includes('word')) return 'doc';
    if (lowerName.match(/\.(mp4|avi|mov|mkv)$/) || mimeType?.includes('video')) return 'video';
    return 'text';
  }
}
