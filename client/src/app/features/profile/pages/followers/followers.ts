import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslatePipe } from '@ngx-translate/core';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile-followers',
  imports: [TranslatePipe],
  templateUrl: './followers.html',
  styleUrl: './followers.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileFollowersPage {
  protected store = inject(ProfileStore);
}
