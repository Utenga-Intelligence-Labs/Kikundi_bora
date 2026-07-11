// Role-based navigation & permissions for Kikundi.
import { Home, Users, PiggyBank, Banknote, Receipt, FileBarChart2, User as UserIcon, ShieldCheck, Wallet, ClipboardList, UserPlus, Heart, Settings, Clock, Activity, FileCheck } from "lucide-react";
import type { Jukumu, User } from "@/api/types";

export interface NavItem {
  to: string;
  label: string;
  icon: any;
}

// Sidebar (desktop) — full menu per role
export const sidebarNav: Record<Jukumu, NavItem[]> = {
  Mwenyekiti: [
    { to: "/dashibodi", label: "Dashibodi", icon: Home },
    { to: "/kamati-mikopo", label: "Kamati ya Mikopo", icon: UserPlus },
    { to: "/mfuko-kijamii", label: "Mfuko wa Kijamii", icon: Heart },
    { to: "/wanachama", label: "Wanachama", icon: Users },
    { to: "/mikopo", label: "Idhinisha Mikopo", icon: ShieldCheck },
    { to: "/vitendo-vinavyosubiri", label: "Vitendo Vinavyosubiri", icon: FileCheck },
    { to: "/michango", label: "Michango", icon: PiggyBank },
    { to: "/marejesho", label: "Marejesho", icon: Receipt },
    { to: "/ripoti", label: "Ripoti", icon: FileBarChart2 },
  ],
  "Mweka Hazina": [
    { to: "/dashibodi", label: "Dashibodi", icon: Home },
    { to: "/kamati-mikopo", label: "Kamati ya Mikopo", icon: UserPlus },
    { to: "/mfuko-kijamii", label: "Mfuko wa Kijamii", icon: Heart },
    { to: "/michango", label: "Pokea Michango", icon: PiggyBank },
    { to: "/marejesho", label: "Pokea Marejesho", icon: Receipt },
    { to: "/mikopo", label: "Mikopo", icon: Wallet },
    { to: "/wanachama", label: "Wanachama", icon: Users },
    { to: "/ripoti", label: "Ripoti za Fedha", icon: FileBarChart2 },
  ],
  Katibu: [
    { to: "/dashibodi", label: "Dashibodi", icon: Home },
    { to: "/kamati-mikopo", label: "Kamati ya Mikopo", icon: UserPlus },
    { to: "/mfuko-kijamii", label: "Mfuko wa Kijamii", icon: Heart },
    { to: "/wanachama", label: "Sajili Wanachama", icon: ClipboardList },
    { to: "/wanachama-kusubiri", label: "Wanaosubiri", icon: Clock },
    { to: "/michango", label: "Kumbukumbu", icon: PiggyBank },
    { to: "/mikopo", label: "Mikopo", icon: Banknote },
    { to: "/ripoti", label: "Ripoti", icon: FileBarChart2 },
  ],
  Mwanachama: [
    { to: "/dashibodi", label: "Akaunti Yangu", icon: Home },
    { to: "/mfuko-kijamii", label: "Mfuko wa Kijamii", icon: Heart },
    { to: "/michango", label: "Michango Yangu", icon: PiggyBank },
    { to: "/mikopo", label: "Mikopo Yangu", icon: Banknote },
    { to: "/marejesho", label: "Marejesho Yangu", icon: Receipt },
  ],
  Msimamizi: [
    { to: "/dashibodi", label: "Dashibodi", icon: Home },
    { to: "/admin", label: "Simamia Watumiaji", icon: Settings },
    { to: "/admin-logs", label: "Kumbukumbu za Mfumo", icon: Activity },
  ],
};

// Returns sidebar nav with committee item injected for ordinary committee members
export function getSidebarNav(jukumu: Jukumu, isCommitteeMember: boolean): NavItem[] {
  const items = [...sidebarNav[jukumu]];
  if (jukumu === "Mwanachama" && isCommitteeMember) {
    // Insert Kamati ya Mikopo after the first item (Dashibodi)
    items.splice(1, 0, { to: "/kamati-mikopo", label: "Kamati ya Mikopo", icon: UserPlus });
  }
  return items;
}

// Mobile bottom nav — keep 5 max, last is profile
export function mobileNav(jukumu: Jukumu, isCommitteeMember: boolean): NavItem[] {
  const items = getSidebarNav(jukumu, isCommitteeMember).slice(0, 4);
  return [...items, { to: "/wasifu", label: "Wasifu", icon: UserIcon }];
}

export const roleSubtitle: Record<Jukumu, string> = {
  Mwenyekiti: "Idhinisha mikopo na fuatilia kikundi",
  "Mweka Hazina": "Simamia fedha, michango na marejesho",
  Katibu: "Kumbukumbu na usajili wa wanachama",
  Mwanachama: "Akiba, mikopo na malipo yako",
  Msimamizi: "Simamia mfumo wote na watumiaji",
};
