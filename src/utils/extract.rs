use anyhow::Result;
use std::fs;
use std::path::Path;

pub fn extract_archive(archive_path: &Path, extract_dir: &Path) -> Result<()> {
    let archive_str = archive_path.to_string_lossy();
    
    if archive_str.ends_with(".tar.gz") {
        extract_tar_gz(archive_path, extract_dir)?;
    } else if archive_str.ends_with(".zip") {
        extract_zip(archive_path, extract_dir)?;
    } else {
        return Err(anyhow::anyhow!("Unsupported archive format"));
    }
    
    Ok(())
}

fn extract_tar_gz(archive_path: &Path, extract_dir: &Path) -> Result<()> {
    let file = fs::File::open(archive_path)?;
    let decompressed = flate2::read::GzDecoder::new(file);
    let mut archive = tar::Archive::new(decompressed);
    
    archive.unpack(extract_dir)?;
    
    Ok(())
}

fn extract_zip(archive_path: &Path, extract_dir: &Path) -> Result<()> {
    let file = fs::File::open(archive_path)?;
    let mut archive = zip::ZipArchive::new(file)?;
    
    for i in 0..archive.len() {
        let mut file = archive.by_index(i)?;
        let outpath = extract_dir.join(file.name());
        
        if file.name().ends_with('/') {
            fs::create_dir_all(&outpath)?;
        } else {
            if let Some(parent) = outpath.parent() {
                fs::create_dir_all(parent)?;
            }
            let mut outfile = fs::File::create(&outpath)?;
            std::io::copy(&mut file, &mut outfile)?;
        }
    }
    
    Ok(())
}
