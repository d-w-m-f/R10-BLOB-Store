import { Component, Input, OnChanges, SimpleChanges, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ManagementService, Worker } from '../../../core/services/management.service';

@Component({
  selector: 'app-daemon-visibility',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './daemon-visibility.component.html',
  styleUrls: ['./daemon-visibility.component.scss']
})
export class DaemonVisibilityComponent implements OnChanges, OnInit {
  @Input() refresh = 0;
  workers: Worker[] = [];

  constructor(private mgtService: ManagementService) {}

  ngOnInit(): void {
    this.loadWorkers();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['refresh']) {
      this.loadWorkers();
    }
  }

  loadWorkers() {
    this.mgtService.getWorkers().subscribe(data => {
      this.workers = data;
    });
  }
}
