import { ChangeDetectionStrategy, Component, inject, OnInit } from '@angular/core';
import { DatePipe } from '@angular/common';
import { TranslatePipe } from '@ngx-translate/core';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile-apps',
  imports: [TranslatePipe],
  templateUrl: './apps.html',
  styleUrl: './apps.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileAppsPage implements OnInit {
  protected store = inject(ProfileStore);

  public ngOnInit(): void {
    this.store.loadHammerheadConnection();
  }
}
