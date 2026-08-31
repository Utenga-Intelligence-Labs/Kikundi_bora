/**
 * Role-switch toggle component.
 * Shows a dropdown/toggle when user has multiple roles.
 * Used in the sidebar near the profile section.
 */

import { useRoleSwitch } from "@/lib/role-context";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { Jukumu } from "@/api/types";
import { Crown, Users } from "lucide-react";

const roleIcons: Record<Jukumu, React.ComponentType<{ className?: string }>> = {
  Mwenyekiti: Crown,
  "Mweka Hazina": Users,
  Katibu: Users,
  Mwanachama: Users,
  Msimamizi: Crown,
};

const roleColors: Record<Jukumu, string> = {
  Mwenyekiti: "bg-purple-500/10 text-purple-700 border-purple-200",
  "Mweka Hazina": "bg-blue-500/10 text-blue-700 border-blue-200",
  Katibu: "bg-amber-500/10 text-amber-700 border-amber-200",
  Mwanachama: "bg-green-500/10 text-green-700 border-green-200",
  Msimamizi: "bg-red-500/10 text-red-700 border-red-200",
};

export function RoleSwitchToggle() {
  const { currentRole, availableRoles, switchRole } = useRoleSwitch();

  // Don't show toggle if only one role
  if (availableRoles.length <= 1 || !currentRole) {
    return null;
  }

  const Icon = roleIcons[currentRole];

  return (
    <div className="px-3 py-3 border-t border-border" data-testid="role-switch-toggle">
      <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground px-3 block mb-2">
        Jukumu
      </label>
      <Select value={currentRole} onValueChange={(role) => switchRole(role as Jukumu)}>
        <SelectTrigger className={`w-full text-sm font-medium border ${roleColors[currentRole]}`}>
          <div className="flex items-center gap-2">
            <Icon className="h-4 w-4" />
            <SelectValue />
          </div>
        </SelectTrigger>
        <SelectContent>
          {availableRoles.map((role) => {
            const RoleIcon = roleIcons[role];
            return (
              <SelectItem key={role} value={role}>
                <div className="flex items-center gap-2">
                  <RoleIcon className="h-4 w-4" />
                  <span>{role}</span>
                </div>
              </SelectItem>
            );
          })}
        </SelectContent>
      </Select>
      <p className="text-xs text-muted-foreground px-3 mt-2">
        {currentRole === "Mwanachama" ? "Mwonekano wa Mwanachama" : "Mwonekano wa Uongozi"}
      </p>
    </div>
  );
}
