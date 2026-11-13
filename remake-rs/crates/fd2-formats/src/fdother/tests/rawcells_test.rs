//! 移植自 `rawcells_test.go`。

use super::original_entry;
use crate::fdother::*;

#[test]
fn parse_raw_cell_literal_transparent_payload() {
    let cell = parse_raw_cell(&[3, 0, 1, 0, 1, 0, 2]).unwrap();
    assert_eq!(cell, RawCell { width: 3, height: 1, pixels: vec![1, 0, 2] });
    let mut dst = vec![9u8; 5];
    cell.blit_at(&mut dst, 5, 1, 0).unwrap();
    assert_eq!(dst, [9, 1, 9, 2, 9]);
}

#[test]
fn parse_raw_cell_rejects_truncated_payload() {
    for raw in [&[][..], &[1, 0, 0, 0], &[2, 0, 1, 0, 7]] {
        assert!(parse_raw_cell(raw).is_err(), "malformed raw cell {raw:?} was accepted");
    }
}

fn raw_cell_bank(cells: &[Vec<u8>]) -> Vec<u8> {
    let directory_end = cells.len() * 4;
    let mut data = vec![0u8; directory_end];
    let mut off = directory_end;
    for (i, cell) in cells.iter().enumerate() {
        data[4 * i..4 * i + 4].copy_from_slice(&(off as u32).to_le_bytes());
        off += cell.len();
    }
    for cell in cells {
        data.extend_from_slice(cell);
    }
    data
}

fn raw_cell(width: u16, height: u16, pixels: &[u8]) -> Vec<u8> {
    let mut data = width.to_le_bytes().to_vec();
    data.extend_from_slice(&height.to_le_bytes());
    data.extend_from_slice(pixels);
    data
}

#[test]
fn raw_cell_bank_preserves_transparent_pixels() {
    let cells = parse_raw_cell_bank(&raw_cell_bank(&[raw_cell(3, 1, &[1, 0, 2]), raw_cell(1, 1, &[7])])).unwrap();
    assert!(cells.len() == 2 && cells[0].width == 3 && cells[1].pixels[0] == 7, "cells={cells:?}");
    let mut dst = vec![9u8; 8];
    cells[0].blit_at(&mut dst, 8, 2, 0).unwrap();
    assert_eq!(&dst[..6], &[9, 9, 1, 9, 2, 9]);
}

#[test]
fn raw_cell_bank_rejects_malformed_geometry() {
    assert!(parse_raw_cell_bank(&raw_cell_bank(&[raw_cell(2, 2, &[1])])).is_err(), "truncated raw cell was accepted");
}

#[test]
fn raw_cell_opaque_offset_writes_literal_zero() {
    let mut dst = vec![9u8; 6];
    let cell = RawCell { width: 2, height: 1, pixels: vec![0, 7] };
    cell.blit_opaque_at_offset(&mut dst, 3, 1).unwrap();
    assert_eq!(dst, [9, 0, 7, 9, 9, 9]);
}

#[test]
fn fdother_002_action_overlay_cells() {
    let Some(data) = original_entry("FDOTHER.DAT", 2) else { return };
    let cells = parse_raw_cell_bank(&data).unwrap();
    assert_eq!(cells.len(), 78, "FDOTHER#2 cell count");
    let mut short = 0;
    for (i, cell) in cells.iter().enumerate() {
        assert!(
            cell.width == 24
                && (cell.height == 20 || cell.height == 16)
                && cell.pixels.len() as i64 == cell.width * cell.height,
            "FDOTHER#2 cell {i}={}x{}",
            cell.width,
            cell.height
        );
        if cell.height == 16 {
            short += 1;
        }
        cell.blit_at(&mut vec![0u8; 320 * 200], 320, 0, 0).unwrap_or_else(|e| panic!("FDOTHER#2 cell {i} blit: {e}"));
    }
    assert_eq!(short, 4, "FDOTHER#2 24x16 cell count");
}
