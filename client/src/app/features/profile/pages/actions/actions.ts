import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { TranslatePipe } from '@ngx-translate/core';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile-actions',
  imports: [TranslatePipe, AppIcon],
  templateUrl: './actions.html',
  styleUrl: './actions.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileActionsPage {
  protected store = inject(ProfileStore);
}
