import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, from, of, concatMap, toArray, map, tap, catchError } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface UploadState {
  fileName: string;
  progress: number;
  status: 'uploading' | 'completed' | 'error';
}

@Injectable({
  providedIn: 'root'
})
export class UploadService {
  private http = inject(HttpClient);
  // Default to 8MB slices to match backend expectations perfectly
  private readonly SLICE_SIZE = 8 * 1024 * 1024;
  // API URL dynamically fetched from environments
  private readonly API_URL = environment.uploadApiUrl;

  /**
   * Orchestrates the chunked upload protocol
   */
  uploadFile(file: File, progressCallback: (progress: number) => void): Observable<any> {
    return this.initUpload(file).pipe(
      concatMap((initRes: { upload_id: string }) => {
        const uploadId = initRes.upload_id;
        const totalChunks = Math.ceil(file.size / this.SLICE_SIZE);
        
        // Create an array of chunk indices
        const chunks = Array.from({ length: totalChunks }, (_, i) => i);
        
        let uploadedChunks = 0;

        // Process chunks sequentially (concatMap ensures order)
        // If we want parallel uploads later, we can use mergeMap with a concurrency limit
        return from(chunks).pipe(
          concatMap(chunkIndex => {
            const start = chunkIndex * this.SLICE_SIZE;
            const end = Math.min(start + this.SLICE_SIZE, file.size);
            const blobSlice = file.slice(start, end);
            
            return this.uploadPart(uploadId, chunkIndex + 1, blobSlice).pipe(
              tap(() => {
                uploadedChunks++;
                const percentDone = Math.round((uploadedChunks / totalChunks) * 100);
                progressCallback(percentDone);
              })
            );
          }),
          // Wait for all chunks to finish
          toArray(),
          // Finally, complete the upload
          concatMap(() => this.completeUpload(uploadId))
        );
      })
    );
  }

  private initUpload(file: File): Observable<{ upload_id: string }> {
    return this.http.post<{ upload_id: string }>(`${this.API_URL}/init`, {
      filename: file.name,
      total_size: file.size,
      content_type: file.type
    });
  }

  private uploadPart(uploadId: string, partNumber: number, blob: Blob): Observable<any> {
    // We send raw binary in the body instead of FormData to save overhead
    return this.http.put(`${this.API_URL}/${uploadId}/parts/${partNumber}`, blob, {
      headers: {
        'Content-Type': 'application/octet-stream'
      }
    });
  }

  private completeUpload(uploadId: string): Observable<any> {
    return this.http.post(`${this.API_URL}/${uploadId}/complete`, {});
  }
}
