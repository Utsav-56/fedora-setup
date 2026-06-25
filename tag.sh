# first tag the version name
read -p "Enter new tag version without (V): " tag

# Now make a .version-info file where the version name is the tag and the date is stored

# the date in this format DD-MM-YYYY hh:mm:ss
CURRENT_DATE=$(date +"%d-%m-%Y %H:%M:%S")
echo "v$tag" > .version-info
echo "$CURRENT_DATE" >> .version-info

echo "Enter commit message for latest changes: "
read -p "" commit_msg

# now commit the changes
git add .
git commit -m "$commit_msg"
git push origin main

git tag "v$tag"
git push origin "v$tag"