import { Component, Input, OnChanges, SimpleChanges } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ManagementService, ClusterStats } from '../../../core/services/management.service';

@Component({
  selector: 'app-monitoring-dashboard',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './monitoring-dashboard.component.html',
  styleUrls: ['./monitoring-dashboard.component.scss']
})
export class MonitoringDashboardComponent implements OnChanges {
  @Input() refresh = 0;
  stats: ClusterStats | null = null;

  constructor(private mgtService: ManagementService) {}

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['refresh']) {
      this.loadStats();
    }
  }

  loadStats() {
    this.mgtService.getClusterStats().subscribe(data => {
      this.stats = data;
    });
  }

  getCapacityPercentage(): number {
    if (!this.stats || this.stats.capacity_mb === 0) return 0;
    return (this.stats.used_mb / this.stats.capacity_mb) * 100;
  }
}
