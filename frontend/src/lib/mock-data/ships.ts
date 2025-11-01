import { Ship } from "@/types/trading";

export const ships: Ship[] = [
  { type_id: 648, name: "Badger", cargo_capacity: 15000 },
  { type_id: 649, name: "Tayra", cargo_capacity: 18500 },
  { type_id: 650, name: "Bestower", cargo_capacity: 19000 },
  { type_id: 651, name: "Wreathe", cargo_capacity: 16500 },
  { type_id: 652, name: "Iteron Mark V", cargo_capacity: 27500 },
  { type_id: 653, name: "Mammoth", cargo_capacity: 19500 },
  { type_id: 654, name: "Sigil", cargo_capacity: 18500 },
  { type_id: 655, name: "Hoarder", cargo_capacity: 17500 },
];

export const authenticatedShips: Ship[] = [
  { type_id: 648, name: "Badger (Your Ship)", cargo_capacity: 15000 },
  { type_id: 649, name: "Tayra (Your Ship)", cargo_capacity: 18500 },
  { type_id: 652, name: "Iteron Mark V (Your Ship)", cargo_capacity: 27500 },
];
